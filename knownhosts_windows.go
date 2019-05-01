package main

import (
	"golang.org/x/crypto/ssh"
)

func KnownHostsHandler() ssh.HostKeyCallback {
	// todo
	return ssh.InsecureIgnoreHostKey()
}
