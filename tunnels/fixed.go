package tunnels

import (
	"io"
	"net"
)

type FixedTunnel struct {
	name     string
	listener net.Listener
	manager  *Manager
	shutdown chan struct{}
	Local    Endpoint
	Remote   Endpoint
}

func (t *FixedTunnel) Run() {
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

func (t *FixedTunnel) Close() {
	if t.listener != nil {
		t.listener.Close()
		t.manager.unregister <- t
		close(t.shutdown)
		t.listener = nil
	}
}

func (t *FixedTunnel) forward(localConn net.Conn) {
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

func (t *FixedTunnel) Name() string {
	return t.name
}
