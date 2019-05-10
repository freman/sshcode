package sessions

import "golang.org/x/crypto/ssh"

type Message interface {
	actOnSession(*Session)
}

type ResizeMessage struct {
	NewHeight int
	NewWidth  int
}

func (t ResizeMessage) actOnSession(s *Session) {
	s.session.WindowChange(t.NewHeight, t.NewWidth)
}

type SigMessage struct {
	Signal ssh.Signal
}

func (t SigMessage) actOnSession(s *Session) {
	s.session.Signal(t.Signal)
}
