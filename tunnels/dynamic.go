package tunnels

import (
	"bytes"
	"encoding/binary"
	"io"
	"log"
	"net"
)

type DynamicTunnel struct {
	name     string
	listener net.Listener
	manager  *Manager
	shutdown chan struct{}
	Local    Endpoint
}

func (t *DynamicTunnel) Run() {
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

func (t *DynamicTunnel) Close() {
	if t.listener != nil {
		t.listener.Close()
		t.manager.unregister <- t
		close(t.shutdown)
		t.listener = nil
	}
}

func (t *DynamicTunnel) forward(localConn net.Conn) {
	defer localConn.Close()

	buf := make([]byte, 256)
	n, err := localConn.Read(buf)
	if err != nil || n < 2 {
		log.Printf("[%s] unable to read SOCKS header: %v", localConn.RemoteAddr(), err)
		return
	}
	buf = buf[:n]

	switch version := buf[0]; version {
	case 4:
		t.socks4(buf, localConn)
	case 5:
		t.socks5(buf, localConn)
	default:
		log.Printf("[%s] unknown SOCKS version: %d", localConn.RemoteAddr(), version)
	}

	<-t.shutdown
}

func (t *DynamicTunnel) Name() string {
	return t.name
}

func (t *DynamicTunnel) socks4(buf []byte, localConn net.Conn) {
	switch command := buf[1]; command {
	case 1:
		port := binary.BigEndian.Uint16(buf[2:4])
		ip := net.IP(buf[4:8])
		addr := &net.TCPAddr{IP: ip, Port: int(port)}
		buf := buf[8:]
		i := bytes.Index(buf, []byte{0})
		if i < 0 {
			log.Printf("[%s] unable to locate SOCKS4 user", localConn.RemoteAddr())
			return
		}
		user := buf[:i]
		log.Printf("[%s] incoming SOCKS4 TCP/IP stream connection, user=%q, raddr=%s", localConn.RemoteAddr(), user, addr)
		remoteConn, err := t.manager.client.DialTCP("tcp", localConn.RemoteAddr().(*net.TCPAddr), addr)
		if err != nil {
			log.Printf("[%s] unable to connect to remote host: %v", localConn.RemoteAddr(), err)
			localConn.Write([]byte{0, 0x5b, 0, 0, 0, 0, 0, 0})
			return
		}
		localConn.Write([]byte{0, 0x5a, 0, 0, 0, 0, 0, 0})
		go io.Copy(localConn, remoteConn)
		go io.Copy(remoteConn, localConn)
	default:
		log.Printf("[%s] unsupported command, closing connection", localConn.RemoteAddr())
	}
}

func (t *DynamicTunnel) socks5(buf []byte, localConn net.Conn) {
	authlen, buf := buf[1], buf[2:]
	auths, buf := buf[:authlen], buf[authlen:]
	if !bytes.Contains(auths, []byte{0}) {
		log.Printf("[%s] unsuported SOCKS5 authentication method", localConn.RemoteAddr())
		localConn.Write([]byte{0x05, 0xff})
		return
	}
	localConn.Write([]byte{0x05, 0x00})
	buf = make([]byte, 256)
	n, err := localConn.Read(buf)
	if err != nil {
		log.Printf("[%s] unable to read SOCKS header: %v", localConn.RemoteAddr(), err)
		return
	}
	buf = buf[:n]
	switch version := buf[0]; version {
	case 5:
		switch command := buf[1]; command {
		case 1:
			buf = buf[3:]
			switch addrtype := buf[0]; addrtype {
			case 1:
				if len(buf) < 8 {
					log.Printf("[%s] corrupt SOCKS5 TCP/IP stream connection request", localConn.RemoteAddr())
					localConn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
					return
				}
				ip := net.IP(buf[1:5])
				port := binary.BigEndian.Uint16(buf[5:6])
				addr := &net.TCPAddr{IP: ip, Port: int(port)}
				log.Printf("[%s] incoming SOCKS5 TCP/IP stream connection, raddr=%s", localConn.RemoteAddr(), addr)
				remoteConn, err := t.manager.client.DialTCP("tcp", localConn.RemoteAddr().(*net.TCPAddr), addr)
				if err != nil {
					log.Printf("[%s] unable to connect to remote host: %v", localConn.RemoteAddr(), err)
					localConn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
					return
				}
				localConn.Write([]byte{0x05, 0x00, 0x00, 0x01, ip[0], ip[1], ip[2], ip[3], byte(port >> 8), byte(port)})
				go io.Copy(localConn, remoteConn)
				go io.Copy(remoteConn, localConn)
			case 3:
				addrlen, buf := buf[1], buf[2:]
				name, buf := buf[:addrlen], buf[addrlen:]
				ip, err := net.ResolveIPAddr("ip", string(name))
				if err != nil {
					log.Printf("[%s] unable to resolve IP address: %q, %v", localConn.RemoteAddr(), name, err)
					localConn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
					return
				}
				port := binary.BigEndian.Uint16(buf[:2])
				addr := &net.TCPAddr{IP: ip.IP, Port: int(port)}
				remoteConn, err := t.manager.client.DialTCP("tcp", localConn.RemoteAddr().(*net.TCPAddr), addr)
				if err != nil {
					log.Printf("[%s] unable to connect to remote host: %v", localConn.RemoteAddr(), err)
					localConn.Write([]byte{0x05, 0x04, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
					return
				}
				localConn.Write([]byte{0x05, 0x00, 0x00, 0x01, addr.IP[0], addr.IP[1], addr.IP[2], addr.IP[3], byte(port >> 8), byte(port)})
				go io.Copy(localConn, remoteConn)
				go io.Copy(remoteConn, localConn)

			default:
				log.Printf("[%s] unsupported SOCKS5 address type: %d", localConn.RemoteAddr(), addrtype)
				localConn.Write([]byte{0x05, 0x08, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
			}
		default:
			log.Printf("[%s] unknown SOCKS5 command: %d", localConn.RemoteAddr(), command)
			localConn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
		}
	default:
		log.Printf("[%s] unnknown version after SOCKS5 handshake: %d", localConn.RemoteAddr(), version)
		localConn.Write([]byte{0x05, 0x07, 0x00, 0x01, 0, 0, 0, 0, 0, 0})
	}

	<-t.shutdown
}
