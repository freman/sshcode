package sessions

import (
	"fmt"

	"golang.org/x/crypto/ssh"
)

type Manager struct {
	client     *ssh.Client
	sessions   map[*Session]struct{}
	broadcast  chan Message
	register   chan *Session
	unregister chan *Session
}

func NewManager(c *ssh.Client) *Manager {
	return &Manager{
		client:     c,
		broadcast:  make(chan Message),
		register:   make(chan *Session),
		unregister: make(chan *Session),
		sessions:   make(map[*Session]struct{}),
	}
}

func (m *Manager) Broadcast(msg Message) {
	select {
	case m.broadcast <- msg:
	default:
		fmt.Println("err sending message")
	}
}

func (m *Manager) NewSession(name string) (*Session, error) {
	sess, err := m.client.NewSession()
	if err != nil {
		return nil, err
	}
	return &Session{
		name:     name,
		manager:  m,
		session:  sess,
		messages: make(chan Message),
	}, nil
}

func (m *Manager) Run() {
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
