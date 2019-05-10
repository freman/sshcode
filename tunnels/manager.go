package tunnels

import (
	"fmt"
	"net"

	"golang.org/x/crypto/ssh"
)

type Manager struct {
	client     *ssh.Client
	tunnels    map[*Tunnel]struct{}
	register   chan *Tunnel
	unregister chan *Tunnel
}

func NewManager(c *ssh.Client) *Manager {
	return &Manager{
		client:     c,
		register:   make(chan *Tunnel),
		unregister: make(chan *Tunnel),
		tunnels:    make(map[*Tunnel]struct{}),
	}
}

func (m *Manager) OpenTunnel(name string, local, remote Endpoint) error {
	listener, err := net.Listen("tcp", local.String())
	if err != nil {
		return err
	}

	local.Port = listener.Addr().(*net.TCPAddr).Port

	tunnel := Tunnel{
		Local:    local,
		Remote:   remote,
		manager:  m,
		listener: listener,
	}

	go tunnel.Run()

	return nil
}

func (m *Manager) Run() {
	for {
		select {
		case tun := <-m.register:
			fmt.Println("[tunnels] Registering " + tun.name)
			m.tunnels[tun] = struct{}{}
		case tun := <-m.unregister:
			fmt.Println("[tunnels] Unegistering " + tun.name)
			if _, ok := m.tunnels[tun]; ok {
				delete(m.tunnels, tun)
				tun.Close()
			}
		}
	}
}
