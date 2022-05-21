package container

import (
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/go-kit/log"
)

type BindMount struct {
	HostPath      string
	ContainerPath string
}

func NewBindMount(hostPath, containerPath string) *BindMount {
	return &BindMount{hostPath, containerPath}
}

// FIXME: Rework all the addr/port stuff so we check for parse errors when creating instead of when e.g converting to int

// e.g 80/tcp
type Port string

func (p *Port) split() (port int, proto string, err error) {
	var (
		parts   = strings.SplitN(string(*p), "/", 2)
		portStr = ""
	)
	proto = "tcp"

	if len(parts) == 2 {
		portStr = parts[0]
		proto = parts[1]
	} else {
		portStr = parts[0]
	}
	port, err = strconv.Atoi(portStr)
	if err != nil {
		return 0, "", fmt.Errorf("port '%s' invalid: %w", portStr, err)
	}
	return port, proto, nil
}
func (p *Port) Number() (int, error) {
	port, _, err := p.split()
	return port, err
}

type Address struct {
	Host string
	Port Port
}

func (a *Address) ProtoAddress() (string, string, error) {
	port, proto, err := a.Port.split()
	if err != nil {
		return "", "", err
	}
	return proto, fmt.Sprintf("%s:%d", a.Host, port), nil
}

type PortMapping map[Port]Address

func (pm *PortMapping) AddPortMapping(containerPort string, hostPort string, hostAddress string) {
	(*pm)[Port(containerPort)] = Address{Host: hostAddress, Port: Port(hostPort)}
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
	PortMapping      *PortMapping
}

type ContainerStatus struct {
	ID          string
	PortMapping *PortMapping
}

type Runner interface {
	Start(*Container) (*ContainerStatus, error)
	LogLogs(id string, logger log.Logger) error
	Stop(id string) error
	Attach(id string) (io.WriteCloser, io.ReadCloser, error)
}
