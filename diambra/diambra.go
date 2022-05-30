package diambra

import (
	"crypto/rand"
	"fmt"
	"io"
	"math/big"
	"os"
	"strings"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/go-kit/log"
	"github.com/go-kit/log/level"

	"github.com/diambra/cli/container"
)

const (
	ContainerPort = "50051/tcp"
)

type Env struct {
	*container.ContainerStatus
	container.Address
}

type Diambra struct {
	log.Logger
	container.Runner
	Envs   []*Env
	config *EnvConfig
}

func NewDiambra(logger log.Logger, config *EnvConfig) (*Diambra, error) {
	runner, err := container.NewDockerRunner(logger, config.AutoRemove)
	if err != nil {
		return nil, fmt.Errorf("couldn't create runner: %w", err)
	}
	if config.PullImage {
		reader, err := runner.PullImage(config.Image)
		if err != nil {
			return nil, fmt.Errorf("couldn't pull image %s: %w", config.Image, err)
		}
		defer reader.Close()
		io.Copy(os.Stderr, reader)
	}
	return &Diambra{
		Logger: logger,
		Runner: runner,
		Envs:   []*Env{},
		config: config,
	}, nil
}

// FIXME: check errors earlier so we don't have to here
func (e *Diambra) EnvsString() (string, error) {
	envs := make([]string, len(e.Envs))
	for i, env := range e.Envs {
		portn, err := env.Port.Number()
		if err != nil {
			return "", fmt.Errorf("invalid port %s: %w", env.Port, err)
		}
		envs[i] = fmt.Sprintf("%s:%d", env.Address.Host, portn)
	}
	return strings.Join(envs, " "), nil
}

// FIXME: Merge with above
func (e *Diambra) EnvsStringContainer() (string, error) {
	portn, err := container.Port(ContainerPort).Number()
	if err != nil {
		return "", err
	}
	envs := make([]string, len(e.Envs))
	for i, env := range e.Envs {
		envs[i] = fmt.Sprintf("%s:%d", env.ContainerStatus.Address, portn)
	}
	return strings.Join(envs, " "), nil
}

func (e *Diambra) waitForGRPC(addr container.Address) error {
	_, hp, err := addr.ProtoAddress()
	if err != nil {
		return err
	}
	for {
		_, err := grpc.Dial(hp, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		if err != nil {
			level.Debug(e.Logger).Log("msg", "couldn't connect to endpoint", "endpoint", hp, "err", err.Error())
			time.Sleep(1 * time.Second)
			continue
		}
		return nil
	}
}
func (d *Diambra) RandInt() (int, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(0xFFFF))
	if err != nil {
		return 0, err
	}
	return int(n.Uint64()), nil
}
func (d *Diambra) Start() error {
	first := true
	agentLogger := d.Logger //e.Screen.NewTab())
	for i := 0; i < d.config.Scale; i++ {
		level.Debug(d.Logger).Log("msg", "creating env container", "envID", i)
		randomSeed, err := d.RandInt()
		if err != nil {
			return fmt.Errorf("couldn't generate random seed: %w", err)
		}
		cs, err := d.Runner.Start(newEnvContainer(d.config, i, randomSeed))
		if err != nil {
			return fmt.Errorf("couldn't start env container: %w", err)
		}
		level.Debug(d.Logger).Log("msg", "started env container", "id", cs.ID)
		env := &Env{
			ContainerStatus: cs,
			Address:         (*cs.PortMapping)[ContainerPort],
		}
		d.Envs = append(d.Envs, env)

		// On first env we wait for the container to start, attach to it until the grpc port is open.
		// This allows diambraEngine to ask for credentials if they don't exist/are expired.
		if first && d.config.Tty {
			wc, rc, err := d.Runner.Attach(cs.ID)
			if err != nil {
				return err
			}
			// Disable stdin -> container when not interactive
			if !d.config.Interactive {
				wc = nil
			}
			streamer := container.NewStreamer(d.Logger, wc, rc)
			if err := streamer.Stream(); err != nil {
				return fmt.Errorf("couldn't attach to container: %w", err)
			}
			level.Debug(d.Logger).Log("msg", "waiting for grpc")
			d.waitForGRPC(env.Address)
			level.Debug(d.Logger).Log("msg", "closing streamer")
			streamer.Close()

			// FIXME: We should just call Render() automatically from the Writer
			/*
				go func() {
					ticker := time.NewTicker(500 * time.Millisecond) // ~30 fps
					for range ticker.C {
						e.Screen.Render()
					}
				}()*/
		}

		go func(id string) {
			level.Debug(d.Logger).Log("msg", "in go func")
			if err := d.Runner.LogLogs(id, log.With(agentLogger, "id", id)); err != nil {
				level.Warn(d.Logger).Log("msg", "LogLogs failed", "err", err.Error())
			}
			level.Debug(d.Logger).Log("msg", "end of go func")
		}(cs.ID)

		level.Debug(d.Logger).Log("msg", "logs copying..")
		first = false
	}
	return nil
}

func newEnvContainer(config *EnvConfig, envID, randomSeed int) *container.Container {
	pm := &container.PortMapping{}
	pm.AddPortMapping(ContainerPort, "0/tcp", "127.0.0.1")

	args := config.AppArgs
	args.RandomSeed = randomSeed
	c := &container.Container{
		Name:        fmt.Sprintf("arena-%3d", envID),
		Image:       config.Image,
		User:        config.User,
		Args:        args.Args(),
		PortMapping: pm,
		BindMounts: []*container.BindMount{
			container.NewBindMount(config.CredPath, "/tmp/.diambraCred"),
			container.NewBindMount(config.RomsPath, "/opt/diambraArena/roms"),
		},
	}
	if config.SeccompProfile != "" {
		c.SecurityOpt = []string{"seccomp=" + config.SeccompProfile}
	}
	return c
}

func (e *Diambra) Cleanup() error {
	var rerr error
	for _, env := range e.Envs {
		level.Debug(e.Logger).Log("msg", "stopping container", "id", env.ContainerStatus.ID)
		if err := e.Runner.Stop(env.ContainerStatus.ID); err != nil {
			rerr = err
			level.Warn(e.Logger).Log("msg", "couldn't stop container", "err", err.Error())
		}
	}
	return rerr
}

func (e *Diambra) StartAgent(image string, args []string) error {
	envs, err := e.EnvsStringContainer()
	if err != nil {
		return err
	}
	c := &container.Container{
		Name:  "agent",
		Image: image,
		Args:  args,
		Env:   []string{"DIAMBRA_ENVS=" + envs},
	}
	cs, err := e.Runner.Start(c)
	if err != nil {
		return err
	}
	wc, rc, err := e.Runner.Attach(cs.ID)
	if err != nil {
		return err
	}
	// Disable stdin -> container when not interactive
	if !e.config.Interactive {
		wc = nil
	}
	streamer := container.NewStreamer(e.Logger, wc, rc)
	if err := streamer.Stream(); err != nil {
		return err
	}
	return nil
}
