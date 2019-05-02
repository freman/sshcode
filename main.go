package main

import (
	"fmt"
	"io"
	"log"
	"net"
	"path"
	"time"

	"github.com/freman/sshcode/authmethod"
	"github.com/google/uuid"
	"github.com/spf13/viper"
	"github.com/zserge/lorca"
	"golang.org/x/crypto/ssh"
)

const codeServerPath = "/tmp/codessh-code-server"

func main() {
	host := flags()
	addr := fmt.Sprintf("%s:%d", host, viper.GetInt("port"))
	login := viper.GetString("login")

	var authMethods []ssh.AuthMethod
	if sshAgent := authmethod.SSHAgent(); sshAgent != nil {
		authMethods = append(authMethods, sshAgent)
	}
	if fileName := viper.GetString("identity"); fileName != "" {
		authMethods = append(authMethods, authmethod.PrivateKeyFile(fileName, authmethod.PromptPassword))
	}

	sshConfig := &ssh.ClientConfig{
		User:            login,
		Auth:            authMethods,
		HostKeyCallback: KnownHostsHandler(),
	}

	connection, err := dial("tcp", addr, sshConfig)
	if err != nil {
		log.Fatalf("Failed to dial: %v", err)
	}

	mgr := newManager(connection)
	go mgr.run()

	upgrade(mgr)

	rand, _ := uuid.NewRandom()
	socketName := "/tmp/code-server." + rand.String() + ".sock"

	session, err := mgr.newSession("code-server")
	if err != nil {
		log.Fatalf("Unable to create session: %v", err)
	}

	go session.run(codeServerPath + " " + viper.GetString("workdir") + " --allow-http --no-auth --socket " + socketName)

	// todo, probe for service status

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		log.Fatal(err)
	}

	defer listener.Close()

	go launchUI(mgr, "http://"+listener.Addr().String())

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Fatal(err)
			}
			go forward(connection, conn, socketName)
		}
	}()

	session.wait()

	cleanup(mgr, socketName)
}

func launchUI(mgr *manager, url string) {
	go func() {
		time.Sleep(5 * time.Second)

		ui, _ := lorca.New(url, "", 480, 320)
		defer ui.Close()

		<-ui.Done()

		mgr.doBroadcast(&sigMessage{signal: ssh.SIGHUP})
	}()
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

func forward(client *ssh.Client, localConn net.Conn, socketName string) {
	remoteConn, err := client.Dial("unix", socketName)
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

func upgrade(mgr *manager) {
	cmd := fmt.Sprintf(`set -euxo pipefail || exit 1; mkdir -p ~/.local/share/code-server; cd %[1]s; /usr/bin/wget -N https://codesrv-ci.cdr.sh/latest-linux; [ -f %[2]s ] && rm %[2]s; ln latest-linux %[2]s; chmod +x %[2]s; exit 0`, path.Dir(codeServerPath), codeServerPath)

	session, err := mgr.newSession("upgrade script")
	if err != nil {
		log.Fatalf("Unable to create session: %v", err)
	}

	if err := session.run(cmd); err != nil {
		log.Fatal("Failed to execute upgrade script: " + err.Error())
	}
}

func cleanup(mgr *manager, socketName string) {
	session, err := mgr.newSession("cleanup")
	if err != nil {
		log.Fatalf("Unable to create session: %v", err)
	}

	if err := session.run("rm " + socketName); err != nil {
		log.Fatal("Failed to execute cleanup script: " + err.Error())
	}
}
