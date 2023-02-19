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

package diambra

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/diambra/cli/pkg/container"
	"github.com/diambra/cli/pkg/diambra/client"
	"github.com/diambra/init/initializer"
	"github.com/go-kit/log"
	"github.com/go-kit/log/level"
	"github.com/spf13/pflag"
)

const (
	DefaultEnvRegistry  = "docker.io"
	DefaultEnvImageName = "diambra/engine"
	DefaultEnvImageTag  = "latest"
)

type AppArgs struct {
	RandomSeed int
	Render     bool
	LockFPS    bool
	Sound      bool
}

type Args []string

func (a *Args) Bool(k string, v bool) {
	if !v {
		return
	}
	*a = append(*a, k)
}
func (a *Args) Int(k string, v int) {
	if v == 0 {
		return
	}
	*a = append(*a, k, strconv.Itoa(v))
}

func (a AppArgs) Args() []string {
	args := Args{}
	args.Bool("--render", a.Render)
	args.Bool("--lockFps", a.LockFPS)
	args.Bool("--sound", a.Sound)
	args.Int("--randomSeed", a.RandomSeed)
	return args
}

type EnvConfig struct {
	logger log.Logger

	AppArgs AppArgs

	Scale      int
	AutoRemove bool
	AgentImage string
	PullImage  bool

	RomsPath string
	CredPath string
	Image    string

	User           string
	SeccompProfile string
	Output         *os.File
	Tty            bool // stdin is a terminal
	Interactive    bool // interaction requested

	Home     string
	Hostname string
	Mounts   []*container.BindMount
	mounts   []string

	PreallocatePort bool

	InitImage string
}

func NewConfig(logger log.Logger) (*EnvConfig, error) {
	userName := ""
	if runtime.GOOS != "windows" {
		u, err := user.Current()
		if err != nil {
			return nil, fmt.Errorf("couldn't get current user: %w", err)
		}
		userName = u.Uid
	}
	homedir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("couldn't get homedir: %w", err)
	}
	hostname, err := os.Hostname()
	if err != nil {
		return nil, fmt.Errorf("couldn't get hostname: %w", err)
	}
	preallocatePort := false
	if runtime.GOOS == "windows" {
		// FIXME: Wrap this in condition that check if runtime is affected
		// by https://github.com/moby/moby/issues/4393
		preallocatePort = true
	}
	return &EnvConfig{
		logger:          logger,
		User:            userName,
		Home:            homedir,
		Hostname:        hostname,
		Output:          os.Stderr,
		PreallocatePort: preallocatePort,
	}, nil
}

func (c *EnvConfig) AddRomsPathFlag(flags *pflag.FlagSet) {
	defaultRomsPath := os.Getenv("DIAMBRAROMSPATH")
	if defaultRomsPath == "" {
		defaultRomsPath = filepath.Join(c.Home, ".diambra", "roms")
	}
	flags.StringVarP(&c.RomsPath, "path.roms", "r", defaultRomsPath, "Path to ROMs (default to DIAMBRAROMSPATH env var if set)")
}

func (c *EnvConfig) AddFlags(flags *pflag.FlagSet) {

	// Path configuration
	flags.StringVar(&c.CredPath, "path.credentials", filepath.Join(c.Home, ".diambra/credentials"), "Path to credentials file")
	c.AddRomsPathFlag(flags)

	// Flags that apply to both agent and env
	flags.BoolVarP(&c.Interactive, "interactive", "i", true, "Open stdin for interactions with arena and agent")
	flags.BoolVarP(&c.PullImage, "images.pull", "p", true, "(Always) pull image before running")

	// Flags to configure env container
	flags.IntVarP(&c.Scale, "env.scale", "s", 1, "Number of environments to run")
	flags.BoolVarP(&c.AutoRemove, "env.autoremove", "x", true, "Remove containers on exit")
	flags.StringVar(&c.Image, "env.image", "", "Env image to use, omit to detect from diambra-arena version")
	flags.StringVar(&c.SeccompProfile, "env.seccomp", "unconfined", "Path to seccomp profile to use for env (may slow down environment). Set to \"\" for runtime's default profile.")
	flags.StringSliceVar(&c.mounts, "env.mount", []string{}, "Host mounts for env container (/host/path:/container/path)")

	// Flags to configure engine in env container
	flags.BoolVarP(&c.AppArgs.Render, "engine.render", "g", false, "Render graphics server side")
	flags.BoolVarP(&c.AppArgs.LockFPS, "engine.lockfps", "l", false, "Lock FPS")
	flags.BoolVarP(&c.AppArgs.Sound, "engine.sound", "n", false, "Enable sound")

	// Agent flags
	flags.StringVarP(&c.AgentImage, "agent.image", "a", "", "Run given agent command in container")

	// Other flags
	flags.StringVar(&c.InitImage, "init.image", "ghcr.io/diambra/init:main", "Init image to use")
}

func (c *EnvConfig) Validate() error {
	exists, isDir := pathExistsAndIsDir(c.RomsPath)
	if !exists {
		return fmt.Errorf("path.roms %s does not exist. Is --path.roms set correctly?", c.RomsPath)
	}
	if !isDir {
		return fmt.Errorf("path.roms %s is not a directory. Is --path.roms set correctly?", c.RomsPath)
	}

	if err := EnsureCredentials(c.logger, c.CredPath); err != nil {
		return err
	}

	if c.Image == "" {
		tag := DefaultEnvImageTag
		parts, err := GetInstalledDiambraArenaVersion()
		if err != nil || len(parts) != 3 || (parts[0] == "0" && parts[1] == "0" && parts[2] == "0") {
			level.Warn(c.logger).Log(
				"msg", "Can't find diambra-arena package to automatically configure env image, using default version. Did you activate your virtual/condaenv?",
				"tag", DefaultEnvImageTag,
				"err", fmt.Sprintf("%v", err),
			)
		} else {
			tag = "v" + strings.Join(parts[:2], ".")
		}
		c.Image = fmt.Sprintf("%s/%s:%s", DefaultEnvRegistry, DefaultEnvImageName, tag)
	}

	c.Mounts = make([]*container.BindMount, len(c.mounts))
	for i, m := range c.mounts {
		p := strings.SplitN(m, ":", 2)
		if len(p) != 2 {
			return fmt.Errorf("invalid mount option %s", m)
		}
		c.Mounts[i] = container.NewBindMount(p[0], p[1])
	}
	return nil
}

func pathExistsAndIsDir(path string) (bool, bool) {
	fi, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return false, false
		}
		panic(err)
	}
	return true, fi.IsDir()
}

type Difficulty string

const (
	DifficultyEasy   Difficulty = "easy"
	DifficultyMedium Difficulty = "medium"
	DifficultyHard   Difficulty = "hard"
)

type SubmissionConfig struct {
	logger log.Logger

	Image        string
	Mode         string
	Difficulty   string
	EnvVars      map[string]string
	Sources      map[string]string
	Secrets      map[string]string
	Command      []string
	ManifestPath string
	SubmissionID int
}

func NewSubmissionConfig(logger log.Logger) *SubmissionConfig {
	return &SubmissionConfig{
		logger: logger,
	}
}

func (c *SubmissionConfig) AddFlags(flags *pflag.FlagSet) {
	flags.StringVar(&c.Mode, "submission.mode", string(client.ModeAIvsCOM), "Mode to use for evaluation")
	flags.StringVar(&c.Difficulty, "submission.difficulty", string(DifficultyEasy), "Difficulty to use for evaluation")
	flags.StringToStringVarP(&c.EnvVars, "submission.env", "e", nil, "Environment variables to pass to the agent")
	flags.StringToStringVarP(&c.Sources, "submission.source", "u", nil, "Source urls to pass to the agent")
	flags.StringToStringVar(&c.Secrets, "submission.secret", nil, "Secrets to pass to the agent")
	flags.StringVar(&c.ManifestPath, "submission.manifest", "", "Path to manifest file.")
	flags.IntVar(&c.SubmissionID, "submission.id", 0, "Submission ID to retrieve manifest from")
}

func (c *SubmissionConfig) Submission(manifest *client.Manifest) (*client.Submission, error) {
	if manifest == nil {
		manifest = &client.Manifest{}
	}
	if c.Image != "" {
		manifest.Image = c.Image
	}
	if c.Mode != "" {
		manifest.Mode = client.Mode(c.Mode)
	}
	if c.Difficulty != "" {
		manifest.Difficulty = c.Difficulty
	}
	if manifest.Image == "" {
		return nil, fmt.Errorf("image is required")
	}

	if c.EnvVars != nil {
		manifest.Env = make(map[string]string)
		for k, v := range c.EnvVars {
			manifest.Env[k] = v
		}
	}

	if c.Sources != nil {
		level.Debug(c.logger).Log("msg", "Using sources", "sources", c.Sources)
		manifest.Sources = make(map[string]string)
		for k, v := range c.Sources {
			manifest.Sources[k] = v
		}
	}

	if c.Command != nil {
		manifest.Command = c.Command
	}

	if manifest.Sources != nil {
		init, err := initializer.NewInitializer(manifest.Sources, c.Secrets)
		if err != nil {
			return nil, err
		}

		if err := init.Validate(); err != nil {
			return nil, err
		}
	}

	return &client.Submission{
		Manifest: *manifest,
		Secrets:  c.Secrets,
	}, nil
}
