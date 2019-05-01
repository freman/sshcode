// +build !windows

package main

import (
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"

	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/knownhosts"
)

func KnownHostsHandler() ssh.HostKeyCallback {
	knownHostsFile := filepath.Join(os.Getenv("HOME"), "/.ssh/known_hosts")

	hosts, err := knownhosts.New(knownHostsFile)
	if err != nil {
		log.Fatal(err)
	}

	return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
		err := hosts(hostname, remote, key)
		switch err := err.(type) {
		case nil:
			// Known host with matching key.
			return nil
		case *knownhosts.KeyError:
			if len(err.Want) == 0 {
				// Unknown host.
				return nil
			}
			head := fmt.Sprintf(`
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
@    WARNING: REMOTE HOST IDENTIFICATION HAS CHANGED!     @
@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@@
IT IS POSSIBLE THAT SOMEONE IS DOING SOMETHING NASTY!
Someone could be eavesdropping on you right now (man-in-the-middle attack)!
It is also possible that a host key has just been changed.
The fingerprint for the %s key sent by the remote host is
%s.
Please contact your system administrator.
Add correct host key in %s to get rid of this message.
`[1:], key.Type(), ssh.FingerprintSHA256(key), knownHostsFile)

			var typeKey *knownhosts.KnownKey
			for i, knownKey := range err.Want {
				if knownKey.Key.Type() == key.Type() {
					typeKey = &err.Want[i]
				}
			}

			var tail string
			if typeKey != nil {
				tail = fmt.Sprintf(
					"Offending %s key in %s:%d",
					typeKey.Key.Type(),
					typeKey.Filename,
					typeKey.Line,
				)
			} else {
				tail = "Host was previously using different host key algorithms:"
				for _, knownKey := range err.Want {
					tail += fmt.Sprintf(
						"\n - %s key in %s:%d",
						knownKey.Key.Type(),
						knownKey.Filename,
						knownKey.Line,
					)
				}
			}
			fmt.Println(head + tail)
		}
		return err
	}
}
