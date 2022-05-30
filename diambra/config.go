package diambra

import (
	"io"
	"strconv"
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
	Tty            bool // stdin is a terminal
	Interactive    bool // interaction requested
	Stdout         io.Writer
	Stderr         io.Writer
}
