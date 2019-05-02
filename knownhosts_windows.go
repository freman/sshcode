package main

import (
	"golang.org/x/crypto/ssh"
)

func KnownHostsHandler() ssh.HostKeyCallback {
	// todo
	// return func(hostname string, remote net.Addr, key ssh.PublicKey) error {
	// 	k, err := registry.OpenKey(registry.CURRENT_USER, `Software\SimonTatham\PuTTY\SshHostKeys`, registry.QUERY_VALUE)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	defer k.Close()

	// 	s, _, err := k.GetStringValue("SystemRoot")
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// 	fmt.Printf("Windows system root is %q\n", s)
	// }
}
