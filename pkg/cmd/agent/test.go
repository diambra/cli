package agent

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	"github.com/containerd/console"
	"github.com/diambra/cli/pkg/container"
	"github.com/diambra/cli/pkg/diambra"
	"github.com/diambra/cli/pkg/diambra/client"
	"github.com/diambra/cli/pkg/log"
	dclient "github.com/docker/docker/client"
	"github.com/go-kit/log/level"
	"github.com/spf13/cobra"
)

const (
	TLSCertPath    = "/etc/ssl/certs"
	EvaluationUser = "1000" // User(ID) that the agent is started as in production
)

func NewTestCmd(logger *log.Logger) *cobra.Command {
	submissionConfig := diambra.NewSubmissionConfig(logger)
	c, err := diambra.NewConfig(logger)
	if err != nil {
		level.Error(logger).Log("msg", err.Error())
		os.Exit(1)
	}

	cmd := &cobra.Command{
		Use:   "test [flags] {--submission.manifest submission-manifest.yaml | docker-image} [args/command(s) ...]",
		Short: "Run an agent from image or manifest similar to how it would be evaluated",
		Long: `This takes a docker image or submission manifest and runs it in the same way as it would be run when submitted
		to DIAMBRA. This is useful for testing your agent before submitting it. Optionally, you can pass in commands to run instead of the configured entrypoint.`,
		Run: func(cmd *cobra.Command, args []string) {
			submission, err := submissionConfig.Submission(c.CredPath, args)
			if err != nil {
				level.Error(logger).Log("msg", "failed to configure manifest", "err", err.Error())
				os.Exit(1)
			}
			if err := TestFn(logger, c, submission); err != nil {
				level.Error(logger).Log("msg", "failed to run agent", "err", err.Error(), "manifest", fmt.Sprintf("%#v", submission.Manifest))
				os.Exit(1)
			}
		},
	}
	c.AddFlags(cmd.Flags())
	submissionConfig.AddFlags(cmd.Flags())
	cmd.Flags().SetInterspersed(false)
	return cmd
}

func TestFn(logger *log.Logger, c *diambra.EnvConfig, submission *client.Submission) error {
	level.Debug(logger).Log("manifest", fmt.Sprintf("%#v", submission.Manifest), "config", fmt.Sprintf("%#v", c))

	client, err := dclient.NewClientWithOpts(dclient.FromEnv, dclient.WithAPIVersionNegotiation())
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
	level.Debug(logger).Log("msg", "starting DIAMBRA env")
	if err := d.Start(); err != nil {
		return fmt.Errorf("could't start DIAMBRA Env: %w", err)
	}

	env := make([]string, len(submission.Manifest.Env))
	i := 0
	for k, v := range submission.Manifest.Env {
		env[i] = fmt.Sprintf("%s=%s", k, v)
		i++
	}
	ctnr := &container.Container{
		Image: submission.Manifest.Image,
		Env:   env,
		User:  EvaluationUser,
	}
	if submission.Manifest.Command != nil {
		ctnr.Command = submission.Manifest.Command
	}
	if submission.Manifest.Args != nil {
		ctnr.Args = submission.Manifest.Args
	}
	if submission.Manifest.Difficulty != "" {
		level.Warn(logger).Log("msg", "difficulty is ignored in test mode")
	}
	if submission.Manifest.Mode != "" {
		level.Warn(logger).Log("msg", "mode is ignored in test mode")
	}
	if submission.Manifest.Sources != nil {
		level.Info(logger).Log("msg", "running init container to fetch sources")
		tmpDir, err := os.MkdirTemp("", "diambra-init")
		if err != nil {
			return fmt.Errorf("couldn't create temp dir: %w", err)
		}
		defer os.RemoveAll(tmpDir)
		sourcesBindMount := container.NewBindMount(tmpDir, "/sources")

		ctnr.BindMounts = []*container.BindMount{sourcesBindMount}

		sourcesJSON, err := json.Marshal(submission.Manifest.Sources)
		if err != nil {
			return fmt.Errorf("failed to marshal sources: %w", err)
		}
		secretsJSON, err := json.Marshal(submission.Secrets)
		if err != nil {
			return fmt.Errorf("failed to marshal secrets: %w", err)
		}

		initContainer := &container.Container{
			Image: c.InitImage,
			BindMounts: []*container.BindMount{
				container.NewBindMount(TLSCertPath, TLSCertPath),
				sourcesBindMount,
			},
			Env: []string{
				"SOURCES=" + string(sourcesJSON),
				"SECRETS=" + string(secretsJSON),
			},
			WorkingDir: "/sources",
			User:       EvaluationUser,
		}
		status, err := d.RunAgentContainer(initContainer)
		if err != nil {
			return fmt.Errorf("failed to run init container: %w", err)
		}
		if status != 0 {
			return fmt.Errorf("init container failed with status %d", status)
		}
	}
	status, err := d.RunAgentContainer(ctnr)
	if err != nil {
		return fmt.Errorf("failed to run agent container: %w", err)
	}
	if status != 0 {
		level.Error(logger).Log("msg", "agent container failed with status", "status", status)
		os.Exit(status)
	}
	return nil
}
