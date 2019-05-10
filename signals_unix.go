// +build !windows

package main

import (
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/freman/sshcode/sessions"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

func signals(mgr *sessions.Manager) {
	signal_chan := make(chan os.Signal, 1)
	signal.Notify(signal_chan,
		syscall.SIGHUP,
		syscall.SIGINT,
		syscall.SIGTERM,
		syscall.SIGQUIT,
		syscall.SIGWINCH)

	go func() {
		for s := range signal_chan {
			switch s {
			case syscall.SIGHUP:
				fmt.Println("main: sighup")
				mgr.Broadcast(&sessions.SigMessage{Signal: ssh.SIGHUP})
			case syscall.SIGINT:
				fmt.Println("main: sigint")
				mgr.Broadcast(&sessions.SigMessage{Signal: ssh.SIGINT})
			case syscall.SIGTERM:
				fmt.Println("main: sigterm")
				mgr.Broadcast(&sessions.SigMessage{Signal: ssh.SIGTERM})
			case syscall.SIGQUIT:
				fmt.Println("main: sigquit")
				mgr.Broadcast(&sessions.SigMessage{Signal: ssh.SIGQUIT})
			case syscall.SIGWINCH:
				fmt.Println("main: sigwinch")
				h, w, err := terminal.GetSize(0)
				if err != nil {
					panic(err)
				}
				mgr.Broadcast(&sessions.ResizeMessage{
					NewHeight: h,
					NewWidth:  w,
				})
			default:
				fmt.Println("I dunno")
			}
		}
	}()
}
