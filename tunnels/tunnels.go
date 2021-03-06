package tunnels

import (
	"fmt"
)

type Endpoint struct {
	Host string
	Port int
}

func (endpoint Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

type Tunnel interface {
	Run()
	Close()
	Name() string
}
