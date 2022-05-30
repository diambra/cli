package diambra

import "io"

type EnvConfig struct {
	LockFPS    bool
	GUI        bool
	Audio      bool
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
