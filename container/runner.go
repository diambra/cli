package container

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

type BindMount struct {
	HostPath      string
	ContainerPath string
}

func NewBindMount(hostPath, containerPath string) *BindMount {
	return &BindMount{hostPath, containerPath}
}

// e.g 80/tcp
type Port string

func (p *Port) split() (port int, proto string, err error) {
	var (
		parts   = strings.SplitN(string(*p), "/", 2)
		portStr = ""
	)
	proto = "TCP"

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

/*
func NewPort(port string) (*Port, error) {
	var (
		p       = strings.SplitN(port, "/", 2)
		portStr = ""
		proto   = "TCP"
	)
	if len(p) == 2 {
		portStr = p[0]
		proto = p[1]
	} else {
		portStr = port
	}

	portn, err := strconv.Atoi(portStr)
	if err != nil {
		return nil, fmt.Errorf("port '%s' invalid: %w", portStr, err)
	}
	return &Port{proto, portn}, nil
}

func (p *Port) String() string {
	return fmt.Sprintf("%d/%s", p.Number, p.Protocol)
}
*/

type Address struct {
	Host string
	Port Port
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
	CopyLogs(id string, stdout, stderr io.Writer) error
	Stop(id string) error
}
