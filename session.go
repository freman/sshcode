package main

import (
	"fmt"
	"os"
	"sync"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
)

type manager struct {
	client     *ssh.Client
	sessions   map[*session]struct{}
	broadcast  chan message
	register   chan *session
	unregister chan *session
}

func newManager(c *ssh.Client) *manager {
	return &manager{
		client:     c,
		broadcast:  make(chan message),
		register:   make(chan *session),
		unregister: make(chan *session),
		sessions:   make(map[*session]struct{}),
	}
}

func (m *manager) doBroadcast(msg message) {
	select {
	case m.broadcast <- msg:
	default:
		fmt.Println("err sending message")
	}
}

func (m *manager) newSession(name string) (*session, error) {
	sess, err := m.client.NewSession()
	if err != nil {
		return nil, err
	}
	return &session{
		name:     name,
		manager:  m,
		session:  sess,
		messages: make(chan message),
	}, nil
}

func (m *manager) run() {
	for {
		select {
		case sess := <-m.register:
			fmt.Println("[manager] Registering " + sess.name)
			m.sessions[sess] = struct{}{}
		case sess := <-m.unregister:
			fmt.Println("[manager] Unegistering " + sess.name)
			if _, ok := m.sessions[sess]; ok {
				delete(m.sessions, sess)
				close(sess.messages)
			}
		case msg := <-m.broadcast:
			fmt.Printf("[manager] Broadcast %T\n", msg)
			for sess := range m.sessions {
				select {
				case sess.messages <- msg:
				default:
					fmt.Println("[manager] Session " + sess.name + " isn't responding, nuking")
					close(sess.messages)
					delete(m.sessions, sess)
				}
			}
		}
	}
}

type session struct {
	name     string
	manager  *manager
	session  *ssh.Session
	messages chan message
	waiting  chan struct{}
	err      error
	mu       sync.Mutex
}

func (s *session) run(cmd string) error {
	s.waiting = make(chan struct{})

	s.manager.register <- s
	defer func() {
		s.manager.unregister <- s
		s.waiting = nil
	}()

	h, w, err := terminal.GetSize(0)
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

func (s *session) wait() {
	if s.waiting != nil {
		<-s.waiting
	}
}

type message interface {
	actOnSession(*session)
}

type resizeMessage struct {
	NewHeight int
	NewWidth  int
}

func (t resizeMessage) actOnSession(s *session) {
	s.session.WindowChange(t.NewHeight, t.NewWidth)
}

type sigMessage struct {
	signal ssh.Signal
}

func (t sigMessage) actOnSession(s *session) {
	s.session.Signal(t.signal)
}
