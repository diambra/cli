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

func (c *EnvConfig) AddFlags(flags *pflag.FlagSet) {
	defaultRomsPath := os.Getenv("DIAMBRAROMSPATH")
	if defaultRomsPath == "" {
		defaultRomsPath = filepath.Join(c.Home, ".diambra", "roms")
	}

	flags.IntVarP(&c.Scale, "scale", "s", 1, "Number of environments to run")
	flags.BoolVarP(&c.AutoRemove, "autoremove", "x", true, "Remove containers on exit")
	flags.BoolVarP(&c.Interactive, "interactive", "i", true, "Open stdin for interactions with arena and agent")

	flags.StringVarP(&c.RomsPath, "romsPath", "r", defaultRomsPath, "Path to ROMs (default to DIAMBRAROMSPATH env var if set)")
	flags.StringVarP(&c.CredPath, "credPath", "c", filepath.Join(c.Home, ".diambraCred"), "Path to credentials file")

	flags.BoolVar(&c.AppArgs.Render, "render", false, "Render graphics server side")
	flags.BoolVar(&c.AppArgs.LockFPS, "lockfps", false, "Lock FPS")
	flags.BoolVar(&c.AppArgs.Sound, "sound", false, "Enable sound")

	flags.BoolVarP(&c.PullImage, "pull", "p", true, "(Always) pull image before running")

	flags.StringVarP(&c.AgentImage, "agent.image", "a", "", "Run agent in container")
	flags.StringVarP(&c.Image, "env.image", "e", DefaultEnvImage, "Env image to use")
	flags.StringVar(&c.SeccompProfile, "env.seccomp", "unconfined", "Path to seccomp profile to use for env (may slow down environment). Set to \"\" for runtime's default profile.")
	flags.StringSliceVar(&c.mounts, "env.mount", []string{}, "Host mounts for env container (/host/path:/container/path)")
}

func (c *EnvConfig) Validate() error {
	exists, isDir := pathExistsAndIsDir(c.RomsPath)
	if !exists {
		return fmt.Errorf("romsPath %s does not exist. Is --romsPath set correctly?", c.RomsPath)
	}
	if !isDir {
		return fmt.Errorf("romsPath %s is not a directory. Is --romsPath set correctly?", c.RomsPath)
	}
	exists, isDir = pathExistsAndIsDir(c.CredPath)
	if exists && isDir {
		return fmt.Errorf("credPath %s is a directory. Is --credPath set correctly?", c.CredPath)
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
