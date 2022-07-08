/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
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
	c, err := diambra.NewConfig()
	if err != nil {
		level.Error(logger).Log("msg", err.Error())
		os.Exit(1)
	}
	fi, err := os.Stdout.Stat()
	if err != nil || (fi.Mode()&os.ModeCharDevice) != 0 {
		c.Tty = true
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Runs a command with DIAMBRA arena started",
		Long: `Run runs the given command after diambraEngine is brought up.
		
It will set the DIAMBRA_ENVS environment variable to list the endpoints of all running environments.

The flag --agent-image can be used to run the commands in the given image.`,
		Args: cobra.MinimumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			level.Debug(logger).Log("config", fmt.Sprintf("%#v", c))
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

	// cmd.LocalFlags().MarkFlagsMutuallyExclusive() Update cobra for this
	return cmd
}

func RunFn(logger *log.Logger, c *diambra.EnvConfig, args []string) error {
	level.Debug(logger).Log("config", fmt.Sprintf("%#v", c))

	//streamer := ui.NewStreamer(logger, os.Stdin, os.Stdout)
	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	runner := container.NewDockerRunner(logger, client, c.AutoRemove)
	console := console.Current()
	d, err := diambra.NewDiambra(logger, console, runner, c) //, streamer)
	if err != nil {
		return fmt.Errorf("couldn't create DIAMBRA Env: %w", err)
	}
	defer d.Cleanup()
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-signalCh
		level.Info(logger).Log("msg", "Received signal, terminating", "signal", s)
		console.Reset()
		// FIXME: Restore terminal
		if err := d.Cleanup(); err != nil {
			level.Error(logger).Log("msg", "cleanup failed", "err", err.Error())
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

	ex := exec.Command(args[0], args[1:]...)
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
