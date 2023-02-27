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

package container

import (
	"fmt"
	"io"
	"os"
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

func (p Port) split() (port int, proto string, err error) {
	var (
		parts   = strings.SplitN(string(p), "/", 2)
		portStr string
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
func (p Port) Number() (int, error) {
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
	Name             string
	Image            string
	Command          []string
	Args             []string
	Env              []string
	User             string
	SecurityOpt      []string
	MemoryLimitBytes int64
	CPULimitCores    float64
	BindMounts       []*BindMount
	PortMapping      *PortMapping
	Hostname         string
	WorkingDir       string
	IPCMode          string

	// If true, the entrypoint of the image will be overridden. Only used for
	// `diambra agent test`.
	OverrideEntrypoint bool
}

type ContainerStatus struct {
	ID          string
	PortMapping *PortMapping
	Address     string
}

type Runner interface {
	Pull(*Container, *os.File) error
	Start(*Container) (*ContainerStatus, error)
	LogLogs(id string, logger log.Logger) error
	Stop(id string) error
	StopAll() error
	Attach(id string) (io.WriteCloser, io.ReadCloser, error)
	Wait(id string) (int, error)
}
