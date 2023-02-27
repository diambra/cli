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

package arena

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/containerd/console"
	"github.com/diambra/cli/pkg/container"
	"github.com/diambra/cli/pkg/diambra"
	"github.com/diambra/cli/pkg/log"
	"github.com/docker/docker/client"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
)

func NewUpCmd(logger *log.Logger) *cobra.Command {
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
		Use:   "up",
		Short: "Start DIAMBRA arena",
		Long:  `This command starts DIAMBRA arena in the background and prints the address for each environment started.`,
		Run: func(cmd *cobra.Command, args []string) {
			level.Debug(logger).Log("config", fmt.Sprintf("%#v", c))
			if err := RunFn(logger, c, args); err != nil {
				if exitErr, ok := err.(*exec.ExitError); ok {
					code := exitErr.ExitCode()
					if code != 0 {
						level.Error(logger).Log("msg", "command failed", "err", err.Error())
					}
					os.Exit(code)
				}
				level.Error(logger).Log("msg", "command failed", "err", err.Error())
				os.Exit(1)
			}
		},
	}

	c.AddFlags(cmd.Flags())

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
	d, err := diambra.NewDiambra(logger, console.Current(), runner, c)
	if err != nil {
		return fmt.Errorf("couldn't create DIAMBRA Env: %w", err)
	}

	level.Debug(logger).Log("msg", "starting DIAMBRA env")
	if err := d.Start(); err != nil {
		return fmt.Errorf("could't start DIAMBRA Env: %w", err)
	}

	envs, err := d.EnvsString()
	if err != nil {
		return err
	}
	fmt.Println(envs)
	return nil
}
