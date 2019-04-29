package authmethod

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"

	"github.com/ScaleFT/sshkeys"
	"github.com/davecgh/go-spew/spew"
	"github.com/howeyc/gopass"
	"golang.org/x/crypto/ssh"
)

func PrivateKeyFile(file string, prompt func(msg string) []byte) ssh.AuthMethod {
	buffer, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}

	var msg = "Enter passphrase for " + file
	var passPhrase []byte
	passPhrasePrompt := func() []byte {
		if len(passPhrase) > 0 {
			return passPhrase
		}
		for len(passPhrase) == 0 {
			passPhrase = prompt(msg)
		}
		return passPhrase
	}

	methods := []func() (ssh.Signer, error){
		func() (ssh.Signer, error) { return ssh.ParsePrivateKey(buffer) },
		func() (ssh.Signer, error) { return ssh.ParsePrivateKeyWithPassphrase(buffer, passPhrasePrompt()) },
		func() (ssh.Signer, error) { return sshkeys.ParseEncryptedPrivateKey(buffer, passPhrasePrompt()) },
	}

retryLoop:
	for {
		for _, method := range methods {
			key, err := method()
			if err != nil {
				if err.Error() == "ssh: cannot decode encrypted private keys" {
					continue
				}
				if err == sshkeys.ErrIncorrectPassword {
					fmt.Fprintln(os.Stderr, "Invalid passphrase")
					passPhrase = []byte{}
					msg = "Bad passphrase, try again for " + file
					continue retryLoop
				}
			}
			return ssh.PublicKeys(key)
		}
		return nil
	}
}

func PromptPassword(msg string) []byte {
	fmt.Print(msg)
	pass, err := gopass.GetPasswd()
	if err != nil {
		spew.Dump(err)
	}
	return bytes.TrimSpace(pass)
}
