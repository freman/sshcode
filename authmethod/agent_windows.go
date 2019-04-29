package authmethod

import (
	"github.com/davidmz/go-pageant"
	"golang.org/x/crypto/ssh"
)

func SSHAgent() ssh.AuthMethod {
	if pageant.Available() {
		return ssh.PublicKeysCallback(pageant.New().Signers)
	}
	return nil
}
