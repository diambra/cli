/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"os/signal"
	"os/user"
	"path/filepath"
	"runtime"
	"syscall"

	"github.com/diambra/cli/diambra"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
)

func pathExists(path string) bool {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return false
		}
		panic(err)
	}
	return true
}

func NewCmdRun() *cobra.Command {
	userName := ""
	if runtime.GOOS != "windows" {
		u, err := user.Current()
		if err != nil {
			level.Error(logger).Log("msg", "couldn't get user", "err", err.Error())
			os.Exit(1)
		}
		userName = u.Uid
	}
	homedir, err := os.UserHomeDir()
	if err != nil {
		level.Error(logger).Log("msg", "couldn't get homedir", "err", err.Error())
		os.Exit(1)
	}
	pipesPath, err := ioutil.TempDir("", "diambra")
	if err != nil {
		level.Error(logger).Log("msg", "couldn't create tempdir", "err", err.Error())
		os.Exit(1)
	}
	c := &diambra.EnvConfig{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		User:   userName,

		PipesPath: pipesPath,
	}
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Runs a command with DIAMBRA arena started",
		Long: `A longer description that spans multiple lines and likely contains examples
and usage of using your command. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
		Run: func(cmd *cobra.Command, args []string) {
			level.Debug(logger).Log("config", fmt.Sprintf("%#v", c))
			if err := RunFn(c, args); err != nil {
				level.Error(logger).Log("msg", "command failed", "err", err.Error())
				if exitErr, ok := err.(*exec.ExitError); ok {
					os.Exit(exitErr.ExitCode())
				}
				os.Exit(1)
			}
		},
	}
	cmd.Flags().BoolVarP(&c.GUI, "gui", "g", true, "Enable GUI")
	cmd.Flags().BoolVarP(&c.LockFPS, "lockfps", "l", true, "Lock FPS")
	cmd.Flags().BoolVarP(&c.Audio, "audio", "a", true, "Enable audio")
	cmd.Flags().IntVarP(&c.Scale, "scale", "s", 1, "Number of environments to run")
	cmd.Flags().StringVarP(&c.RomsPath, "romsPath", "r", filepath.Join(homedir, ".diambra", "roms"), "Path to ROMs")
	cmd.Flags().StringVarP(&c.CredPath, "credPath", "c", filepath.Join(homedir, ".diambraCred"), "Path to credentials file")
	cmd.Flags().SetInterspersed(false)

	// cmd.LocalFlags().MarkFlagsMutuallyExclusive() Update cobra for this
	return cmd
}

func init() {
	rootCmd.AddCommand(NewCmdRun())
}

func RunFn(c *diambra.EnvConfig, args []string) error {
	level.Debug(logger).Log("config", fmt.Sprintf("%#v", c))
	if !pathExists(c.RomsPath) {
		return fmt.Errorf("romsPath %s does not exist. Is --romsPath set correctly?", c.RomsPath)
	}
	if !pathExists(c.CredPath) {
		fh, err := os.OpenFile(c.CredPath, os.O_RDONLY|os.O_CREATE, 0600)
		if err != nil {
			return fmt.Errorf("can't create credentials file %s: %w", c.CredPath, err)
		}
		fh.Close()
	}

	d, err := diambra.NewEnv(logger, c)
	if err != nil {
		return fmt.Errorf("couldn't create DIAMBRA Env: %w", err)
	}
	defer d.Cleanup()
	signalCh := make(chan os.Signal, 1)
	signal.Notify(signalCh, os.Interrupt, syscall.SIGTERM)
	go func() {
		s := <-signalCh
		level.Info(logger).Log("msg", "Received signal, terminating", "signal", s)
		if err := d.Cleanup(); err != nil {
			level.Error(logger).Log("msg", "cleanup failed", "err", err.Error())
		}
		os.Exit(1)
	}()
	level.Debug(logger).Log("msg", "starting DIAMBRA env")
	if err := d.Start(); err != nil {
		return fmt.Errorf("could't start DIAMBRA Env: %w", err)
	}
	level.Debug(logger).Log("msg", "DIAMBRA env started")

	ex := exec.Command(args[0], args[1:]...)
	ex.Env = os.Environ()
	ex.Env = append(ex.Env, "PIPES_PATH="+c.PipesPath)
	ex.Stdin = os.Stdin
	ex.Stdout = os.Stdout
	ex.Stderr = os.Stderr
	level.Debug(logger).Log("msg", "running command", "args", fmt.Sprintf("%#v", args), "env", fmt.Sprintf("%#v", ex.Env))
	return ex.Run()
}
