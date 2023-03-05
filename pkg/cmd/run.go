/*
 * Copyright 2022 The DIAMBRA Authors
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package cmd

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/containerd/console"
	"github.com/diambra/cli/pkg/container"
	"github.com/diambra/cli/pkg/diambra"
	"github.com/diambra/cli/pkg/log"

	"github.com/docker/docker/client"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
)

func NewCmdRun(logger *log.Logger) *cobra.Command {
	c, err := diambra.NewConfig(logger)
	if err != nil {
		level.Error(logger).Log("msg", err.Error())
		os.Exit(1)
	}
	fi, err := os.Stdout.Stat()
	if err != nil || (fi.Mode()&os.ModeCharDevice) != 0 {
		c.Tty = true
	}

	cmd := &cobra.Command{
		Use:   "run [flags] command [args...]",
		Short: "Runs a command with DIAMBRA arena started",
		Long: `Run runs the given command after diambraEngine is brought up.

It will set the DIAMBRA_ENVS environment variable to list the endpoints of all running environments.
The DIAMBRA arena python package will automatically be configured by this.

The flag --agent-image can be used to run the commands in the given image.`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := RunFn(logger, c, args); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					code := exitErr.ExitCode()
					if code != 0 {
						level.Error(logger).Log("msg", "Couldn't run", "err", err.Error())
					}
					os.Exit(code)
				}
				level.Error(logger).Log("msg", "Couldn't run", "err", err.Error())
				os.Exit(1)
			}
		},
	}

	c.AddFlags(cmd.Flags())
	cmd.Flags().SetInterspersed(false)
	return cmd
}

func RunFn(logger *log.Logger, c *diambra.EnvConfig, args []string) error {
	level.Debug(logger).Log("config", fmt.Sprintf("%#v", c))

	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	runner, err := container.NewDockerRunner(logger, client, c.AutoRemove)
	if err != nil {
		return err
	}
	console := console.Current()
	d, err := diambra.NewDiambra(logger, console, runner, c)
	if err != nil {
		return fmt.Errorf("couldn't create DIAMBRA Env: %w", err)
	}
	defer func() {
		if err := d.Cleanup(); err != nil {
			level.Error(logger).Log("msg", "Couldn't cleanup DIAMBRA Env", "err", err.Error())
		}
	}()
	var (
		signalCh = make(chan os.Signal, 1)
		ex       *exec.Cmd
	)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-signalCh
		level.Info(logger).Log("msg", "Received signal, terminating", "signal", s)
		if err := console.Reset(); err != nil {
			level.Error(logger).Log("msg", "Couldn't reset console", "err", err.Error())
		}
		if err := d.Cleanup(); err != nil {
			level.Error(logger).Log("msg", "cleanup failed", "err", err.Error())
		}
		if ex != nil {
			if err := ex.Process.Kill(); err != nil {
				level.Error(logger).Log("msg", "Couldn't kill process", "err", err.Error())
			}
		}
		os.Exit(1)
	}()

	level.Info(logger).Log("msg", "Starting DIAMBRA environment:")
	if err := d.Start(); err != nil {
		return err
	}

	envs, err := d.EnvsString()
	if err != nil {
		return err
	}
	level.Info(logger).Log("msg", "DIAMBRA environment started")

	if c.AgentImage != "" {
		return d.RunAgentImage(c.AgentImage, args)
	}
	if len(args) == 0 {
		return errors.New("command required when not using --agent-image")
	}

	ex = exec.Command(args[0], args[1:]...)
	ex.Env = os.Environ()
	ex.Env = append(ex.Env, fmt.Sprintf("DIAMBRA_ENVS=%s", envs))
	if c.Interactive {
		ex.Stdin = os.Stdin
	}
	ex.Stdout = os.Stdout
	ex.Stderr = os.Stderr
	level.Debug(logger).Log("msg", "running command", "args", fmt.Sprintf("%#v", args), "env", fmt.Sprintf("%#v", ex.Env))
	return ex.Run()
}
