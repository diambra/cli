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
	"github.com/spf13/pflag"
)

const DefaultEnvImage = "diambra/engine:main"

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
}

func NewConfig() (*EnvConfig, error) {
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
	return &EnvConfig{
		User:     userName,
		Home:     homedir,
		Hostname: hostname,
		Output:   os.Stderr,
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
	flags.StringVarP(&c.Image, "env.image", "e", DefaultEnvImage, "Env image to use")
	flags.StringVar(&c.SeccompProfile, "env.seccomp", "unconfined", "Path to seccomp profile to use for env (may slow down environment). Set to \"\" for runtime's default profile.")
	flags.StringSliceVar(&c.mounts, "env.mount", []string{}, "Host mounts for env container (/host/path:/container/path)")

	// Flags to configure engine in env container
	flags.BoolVarP(&c.AppArgs.Render, "engine.render", "g", false, "Render graphics server side")
	flags.BoolVarP(&c.AppArgs.LockFPS, "engine.lockfps", "l", false, "Lock FPS")
	flags.BoolVarP(&c.AppArgs.Sound, "engine.sound", "n", false, "Enable sound")

	// Agent flags
	flags.StringVarP(&c.AgentImage, "agent.image", "a", "", "Run given agent command in container")

}

func (c *EnvConfig) Validate() error {
	exists, isDir := pathExistsAndIsDir(c.RomsPath)
	if !exists {
		return fmt.Errorf("path.roms %s does not exist. Is --path.roms set correctly?", c.RomsPath)
	}
	if !isDir {
		return fmt.Errorf("path.roms %s is not a directory. Is --path.roms set correctly?", c.RomsPath)
	}
	exists, isDir = pathExistsAndIsDir(c.CredPath)
	if exists && isDir {
		return fmt.Errorf("path.credentials %s is a directory. Is --path.credentials set correctly?", c.CredPath)
	}
	if !exists {
		fh, err := os.OpenFile(c.CredPath, os.O_RDONLY|os.O_CREATE, 0600)
		if err != nil {
			return fmt.Errorf("can't create credentials file %s: %w", c.CredPath, err)
		}
		fh.Close()
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
