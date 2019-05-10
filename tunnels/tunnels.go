package tunnels

import (
	"fmt"
	"io"
	"net"
)

type Endpoint struct {
	Host string
	Port int
}

func (endpoint Endpoint) String() string {
	return fmt.Sprintf("%s:%d", endpoint.Host, endpoint.Port)
}

type Tunnel struct {
	name     string
	listener net.Listener
	manager  *Manager
	shutdown chan struct{}
	Local    Endpoint
	Remote   Endpoint
}

func (t *Tunnel) Run() {
	defer t.Close()
	t.shutdown = make(chan struct{})

	for {
		conn, err := t.listener.Accept()
		if err != nil {
			return
		}

		go t.forward(conn)
	}
}

func (t *Tunnel) Close() {
	if t.listener != nil {
		t.listener.Close()
		t.manager.unregister <- t
		close(t.shutdown)
		t.listener = nil
	}
}

func (t *Tunnel) forward(localConn net.Conn) {
	defer localConn.Close()
	remoteConn, err := t.manager.client.Dial("tcp", t.Remote.String())
	if err != nil {
		return
	}
	defer remoteConn.Close()

	go io.Copy(localConn, remoteConn)
	go io.Copy(remoteConn, localConn)

	<-t.shutdown
}
