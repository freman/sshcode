package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"path"
	"time"

	"github.com/google/uuid"
	"github.com/zserge/lorca"

	"github.com/freman/sshcode/authmethod"
	"github.com/spf13/viper"

	"golang.org/x/crypto/ssh"
)

const codeServerPath = "/tmp/codessh-code-server"

func main() {
	host := flags()
	addr := fmt.Sprintf("%s:%d", host, viper.GetInt("port"))
	login := viper.GetString("login")

	authMethods := []ssh.AuthMethod{authmethod.SSHAgent()}
	if fileName := viper.GetString("identity"); fileName != "" {
		authMethods = append(authMethods, authmethod.PrivateKeyFile(fileName, authmethod.PromptPassword))
	}

	sshConfig := &ssh.ClientConfig{
		User:            login,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	connection, err := dial("tcp", addr, sshConfig)
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}

	s := &sshcode{
		client:   connection,
		sessions: make(map[*ssh.Session]struct{}),
	}

	fmt.Println("ensuring code-server is updated...")

	cmd := fmt.Sprintf(`set -euxo pipefail || exit 1; mkdir -p ~/.local/share/code-server; cd %[1]s; /usr/bin/wget -N https://codesrv-ci.cdr.sh/latest-linux; [ -f %[2]s ] && rm %[2]s; ln latest-linux %[2]s; chmod +x %[2]s; exit 0`, path.Dir(codeServerPath), codeServerPath)

	if err := s.runCommand(cmd); err != nil {
		log.Fatal("Unable to execute script: %v", err)
	}

	rand, _ := uuid.NewRandom()
	socketName := "/tmp/code-server." + rand.String() + ".sock"
	go func() {
		fmt.Println(s.runCommand(codeServerPath + " " + viper.GetString("workdir") + " --allow-http --no-auth --socket " + socketName))
	}()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()

	go func() {
		time.Sleep(5 * time.Second)
		url := "http://" + listener.Addr().String()
		ui, _ := lorca.New(url, "", 480, 320)
		defer ui.Close()

		<-ui.Done()

		for session := range s.sessions {
			session.Signal(ssh.SIGHUP)
		}
		for len(s.sessions) > 0 {
			fmt.Println("waiting for shutdown")
			time.Sleep(500 * time.Millisecond)
		}
		s.runCommand("rm " + socketName)
		for len(s.sessions) > 0 {
			fmt.Println("waiting for cleanup")
			time.Sleep(500 * time.Millisecond)
		}

		listener.Close()
	}()

	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Fatal(err)
		}
		go s.forward(conn, socketName)
	}
}

type sshcode struct {
	client   *ssh.Client
	sessions map[*ssh.Session]struct{}
}

func dial(network, addr string, config *ssh.ClientConfig) (*ssh.Client, error) {
	if bindAddr := viper.GetString("bind"); bindAddr != "" {
		tcpAddr, err := net.ResolveTCPAddr("tcp", bindAddr)
		if err != nil {
			return nil, err
		}
		conn, err := (&net.Dialer{LocalAddr: tcpAddr, Timeout: config.Timeout}).Dial("tcp", addr)
		if err != nil {
			return nil, err
		}
		c, chans, reqs, err := ssh.NewClientConn(conn, addr, config)
		if err != nil {
			return nil, err
		}
		return ssh.NewClient(c, chans, reqs), nil
	}
	return ssh.Dial(network, addr, config)
}

func (s *sshcode) runCommand(cmd string) error {
	session, err := s.client.NewSession()
	if err != nil {
		return err
	}
	defer session.Close()
	session.RequestPty(os.Getenv("TERM"), 80, 80, ssh.TerminalModes{
		ssh.ECHO:          0,     // disable echoing
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
	})
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	s.sessions[session] = struct{}{}
	err = session.Run(cmd)
	delete(s.sessions, session)
	return err
}

func (s *sshcode) forward(localConn net.Conn, socketName string) {
	remoteConn, err := s.client.Dial("unix", socketName)
	if err != nil {
		fmt.Printf("Remote dial error: %s\n", err)
		return
	}

	copyConn := func(writer, reader net.Conn) {
		_, err := io.Copy(writer, reader)
		if err != nil {
			fmt.Printf("io.Copy error: %s", err)
		}
	}

	go copyConn(localConn, remoteConn)
	go copyConn(remoteConn, localConn)
}
