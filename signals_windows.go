package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"golang.org/x/crypto/ssh"
)

// Todo, handle window resize

func signals(mgr *manager) {
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
				mgr.doBroadcast(&sigMessage{signal: ssh.SIGHUP})
			case syscall.SIGINT:
				fmt.Println("main: sigint")
				mgr.doBroadcast(&sigMessage{signal: ssh.SIGINT})
			case syscall.SIGTERM:
				fmt.Println("main: sigterm")
				mgr.doBroadcast(&sigMessage{signal: ssh.SIGTERM})
			case syscall.SIGQUIT:
				fmt.Println("main: sigquit")
				mgr.doBroadcast(&sigMessage{signal: ssh.SIGQUIT})
			default:
				fmt.Println("I dunno")
			}
		}
	}()
}
