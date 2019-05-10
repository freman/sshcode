package sessions

import (
	"os"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

type Session struct {
	name     string
	manager  *Manager
	session  *ssh.Session
	messages chan Message
	waiting  chan struct{}
	err      error
	mu       sync.Mutex
}

func (s *Session) Run(cmd string) error {
	s.waiting = make(chan struct{})

	s.manager.register <- s
	defer func() {
		s.manager.unregister <- s
		s.waiting = nil
	}()

	h, w, err := terminal.GetSize(int(os.Stdout.Fd()))
	if err != nil {
		s.err = err
		return err
	}

	s.session.RequestPty(os.Getenv("TERM"), h, w, ssh.TerminalModes{
		ssh.ECHO:          0,
		ssh.TTY_OP_ISPEED: 115200,
		ssh.TTY_OP_OSPEED: 115200,
	})

	s.session.Stdout = os.Stdout
	s.session.Stderr = os.Stderr

	go func() {
		err := s.session.Run(cmd)
		if err != nil && s.err == nil {
			s.err = err
		}
		close(s.waiting)
	}()

	for {
		select {
		case msg, ok := <-s.messages:
			if !ok {
				return s.err
			}
			msg.actOnSession(s)
		case <-s.waiting:
			return s.err
		}
	}

	return s.err
}

func (s *Session) Wait() {
	if s.waiting != nil {
		<-s.waiting
	}
}
