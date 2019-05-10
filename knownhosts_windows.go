package main

import (
	"github.com/freman/putty_hosts"
	"golang.org/x/crypto/ssh"
)

func KnownHostsHandler() ssh.HostKeyCallback {
	cb, err := putty_hosts.KnownHosts()
	if err != nil {
		panic(err)
	}
	return cb
}
