/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package arena

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/diambra/cli/pkg/container"
	"github.com/diambra/cli/pkg/diambra"
	"github.com/diambra/cli/pkg/log"
	"github.com/docker/docker/client"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
)

func NewUpCmd(logger *log.Logger) *cobra.Command {
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

	cmd.Flags().SetInterspersed(false)

	return cmd
}

func RunFn(logger *log.Logger, c *diambra.EnvConfig, args []string) error {
	level.Debug(logger).Log("config", fmt.Sprintf("%#v", c))

	client, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return err
	}
	runner := container.NewDockerRunner(logger, client, c.AutoRemove)

	d, err := diambra.NewDiambra(logger, runner, c)
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
