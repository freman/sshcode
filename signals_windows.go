package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/freman/sshcode/sessions"
	"golang.org/x/crypto/ssh"
)

// Todo, handle window resize

func signals(mgr *sessions.Manager) {
	signal_chan := make(chan os.Signal, 1)
	signal.Notify(signal_chan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT)

	go func() {
		for s := range signal_chan {
			switch s {
			case syscall.SIGHUP:
				fmt.Println("main: sighup")
				mgr.Broadcast(&sessions.SigMessage{signal: ssh.SIGHUP})
			case syscall.SIGINT:
				fmt.Println("main: sigint")
				mgr.Broadcast(&sessions.SigMessage{signal: ssh.SIGINT})
			case syscall.SIGTERM:
				fmt.Println("main: sigterm")
				mgr.Broadcast(&sessions.SigMessage{signal: ssh.SIGTERM})
			case syscall.SIGQUIT:
				fmt.Println("main: sigquit")
				mgr.Broadcast(&sessions.SigMessage{signal: ssh.SIGQUIT})
			default:
				fmt.Println("I dunno")
			}
		}
	}()
}
