package agent

import (
	"encoding/json"
	"fmt"
	"os"

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
	TLSCertPath = "/etc/ssl/certs"
)

func NewTestCmd(logger *log.Logger) *cobra.Command {
	submissionConfig := diambra.NewSubmissionConfig(logger)
	c, err := diambra.NewConfig(logger)
	if err != nil {
		level.Error(logger).Log("msg", err.Error())
		os.Exit(1)
	}

	cmd := &cobra.Command{
		Use:   "test [--submission.manifest submission-manifest.yaml | docker-image]",
		Short: "Run an agent from image or manifest similar to how it would be evaluated",
		Long: `This takes a docker image or submission manifest and runs it in the same way as it would be run when submitted
		to DIAMBRA. This is useful for testing your agent before submitting it.`,
		Args: cobra.MaximumNArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			if len(args) > 0 {
				submissionConfig.Image = args[0]
			}
			submission, err := submissionConfig.Submission()
			if err != nil {
				level.Error(logger).Log("msg", "failed to configure manifest", "err", err.Error())
				os.Exit(1)
			}
			if err := TestFn(logger, c, submission); err != nil {
				level.Error(logger).Log("msg", "failed to run agent", "err", err.Error())
				os.Exit(1)
			}

		},
	}
	c.AddFlags(cmd.Flags())
	submissionConfig.AddFlags(cmd.Flags())
	return cmd
}

func TestFn(logger *log.Logger, c *diambra.EnvConfig, submission *client.Submission) error {
	level.Debug(logger).Log("config", fmt.Sprintf("%#v", c))

	client, err := dclient.NewClientWithOpts(dclient.FromEnv, dclient.WithAPIVersionNegotiation())
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

	env := make([]string, len(submission.Manifest.Env))
	i := 0
	for k, v := range submission.Manifest.Env {
		env[i] = fmt.Sprintf("%s=%s", k, v)
		i++
	}
	ctnr := &container.Container{
		Image: submission.Manifest.Image,
		Env:   env,
	}
	if submission.Manifest.Command != nil {
		ctnr.Args = submission.Manifest.Command
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
