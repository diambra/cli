package container

import "io"

type BindMount struct {
	HostPath      string
	ContainerPath string
}

func NewBindMount(hostPath, containerPath string) *BindMount {
	return &BindMount{hostPath, containerPath}
}

type Container struct {
	Image            string
	Command          []string
	Env              []string
	User             string
	SecurityOpt      []string
	MemoryLimitBytes int64
	CPULimitCores    float64
	BindMounts       []*BindMount
}

type ContainerStatus struct {
	ID string
}

type Runner interface {
	Start(*Container) (*ContainerStatus, error)
	CopyLogs(id string, stdout, stderr io.Writer) error
	Stop(id string) error
}
