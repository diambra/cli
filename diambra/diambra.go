package diambra

import (
	"io"
	"strconv"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/diambra/cli/container"
)

const (
	DefaultEnvImage = "diambraengine:latest"
)

type EnvConfig struct {
	LockFPS bool
	GUI     bool
	Audio   bool
	Scale   int

	PipesPath string
	RomsPath  string
	CredPath  string

	User string

	Stdout io.Writer
	Stderr io.Writer
}

type Env struct {
	log.Logger
	container.Runner
	containers []*container.ContainerStatus
	config     *EnvConfig
}

func NewEnv(logger log.Logger, config *EnvConfig) (*Env, error) {
	runner, err := container.NewDockerRunner(logger)
	if err != nil {
		return nil, err
	}
	return &Env{
		Logger:     logger,
		Runner:     runner,
		containers: []*container.ContainerStatus{},
		config:     config,
	}, nil
}

func (e *Env) Start() error {
	for i := 0; i < e.config.Scale; i++ {
		level.Debug(e.Logger).Log("msg", "creating env container", "envID", i)
		cs, err := e.Runner.Start(newEnvContainer(e.config, i))
		if err != nil {
			return err
		}
		level.Debug(e.Logger).Log("msg", "started env container")
		e.containers = append(e.containers, cs)
		level.Debug(e.Logger).Log("msg", "appended container to list")

		level.Debug(e.Logger).Log("msg", "starting gofunc")
		go func(id string) {
			level.Debug(e.Logger).Log("msg", "in go func")
			e.Runner.CopyLogs(id, e.config.Stdout, e.config.Stderr)
			level.Debug(e.Logger).Log("msg", "end of go func")
		}(e.containers[i].ID)
		level.Debug(e.Logger).Log("msg", "logs copying..")
	}
	level.Debug(e.Logger).Log("msg", "Start() end")
	return nil
}

func newEnvContainer(config *EnvConfig, envID int) *container.Container {
	envIDStr := strconv.Itoa(envID)
	return &container.Container{
		Image:   DefaultEnvImage,
		Command: []string{"--pipesPath", "/pipes", "--envId", envIDStr},
		Env:     []string{"PIPES_PATH=/pipes", "ENV_ID=" + envIDStr},
		User:    config.User,
		BindMounts: []*container.BindMount{
			container.NewBindMount(config.PipesPath, "/pipes"),
			container.NewBindMount(config.CredPath, "/tmp/.diambraCred"),
			container.NewBindMount(config.RomsPath, "/opt/diambraArena/roms"),
		},
	}
}

func (e *Env) Cleanup() error {
	var rerr error
	for _, c := range e.containers {
		level.Debug(e.Logger).Log("msg", "stopping container", "id", c.ID)
		if err := e.Runner.Stop(c.ID); err != nil {
			rerr = err
			level.Warn(e.Logger).Log("msg", "couldn't stop container", "err", err.Error())
		}
	}
	return rerr
}
