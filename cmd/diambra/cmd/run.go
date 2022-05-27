/*
Copyright © 2022 NAME HERE <EMAIL ADDRESS>

*/
package cmd

import (
	"fmt"
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

const DefaultEnvImage = "diambra/engine:main"

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
	c := &diambra.EnvConfig{
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		User:   userName,
	}

	fi, err := os.Stdout.Stat()
	if err != nil || (fi.Mode()&os.ModeCharDevice) != 0 {
		c.Interactive = true
	}

	// FIXME: Is the current working directory a good option for this?
	if c.RunID == "" {
		wd, err := os.Getwd()
		if err != nil {
			level.Error(logger).Log("msg", "couldn't get current directory and --run-id is not set", "err", err.Error())
			os.Exit(1)
		}
		c.RunID = wd
	}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Runs a command with DIAMBRA arena started",
		Long: `Run runs the given command after diambraEngine is brought up.
		
It will set the DIAMBRA_ENVS environment variable to list the endpoints of all running environments`,
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
	defaultRomsPath := os.Getenv("DIAMBRAROMSPATH")
	if defaultRomsPath == "" {
		defaultRomsPath = filepath.Join(homedir, ".diambra", "roms")
	}
	cmd.Flags().BoolVarP(&c.GUI, "gui", "g", true, "Enable GUI")
	cmd.Flags().BoolVarP(&c.LockFPS, "lockfps", "l", true, "Lock FPS")
	cmd.Flags().BoolVarP(&c.Audio, "audio", "a", true, "Enable audio")
	cmd.Flags().StringVarP(&c.AgentImage, "agent-image", "i", "", "Run agent in container")
	cmd.Flags().BoolVarP(&c.AutoRemove, "autoremove", "x", true, "Remove containers on exit")
	cmd.Flags().BoolVarP(&c.PullImage, "pull", "p", true, "(Always) pull image before running")
	cmd.Flags().StringVarP(&c.RunID, "run-id", "u", "", "(Always) pull image before running")

	cmd.Flags().IntVarP(&c.Scale, "scale", "s", 1, "Number of environments to run")
	cmd.Flags().StringVarP(&c.RomsPath, "romsPath", "r", defaultRomsPath, "Path to ROMs")
	cmd.Flags().StringVarP(&c.CredPath, "credPath", "c", filepath.Join(homedir, ".diambraCred"), "Path to credentials file")
	cmd.Flags().StringVarP(&c.Image, "image", "e", DefaultEnvImage, "Env image to use")

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

	d, err := diambra.NewDiambra(logger, c)
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

	envs, err := d.EnvsString()
	if err != nil {
		return err
	}
	level.Debug(logger).Log("msg", "DIAMBRA env started")

	if c.AgentImage != "" {
		return d.StartAgent(c.AgentImage, args)
	}
	ex := exec.Command(args[0], args[1:]...)
	ex.Env = os.Environ()
	ex.Env = append(ex.Env, fmt.Sprintf("DIAMBRA_ENVS=%s", envs))
	ex.Stdin = os.Stdin
	ex.Stdout = os.Stdout
	ex.Stderr = os.Stderr
	level.Debug(logger).Log("msg", "running command", "args", fmt.Sprintf("%#v", args), "env", fmt.Sprintf("%#v", ex.Env))
	return ex.Run()
}
