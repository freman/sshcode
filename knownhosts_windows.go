package main

import (
	"github.com/freman/putty_hosts"
	"golang.org/x/crypto/ssh"
)

func KnownHostsHandler() ssh.HostKeyCallback {
	return putty_hosts.KnownHosts()
}
