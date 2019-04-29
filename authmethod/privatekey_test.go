package authmethod_test

import (
	"path/filepath"
	"testing"

	"github.com/freman/sshcode/authmethod"
)

func TestPrivateKeyFile(t *testing.T) {
	t.Parallel()

	for _, file := range []string{"testnopass", "testpass"} {
		file := file // capture range variable
		t.Run(file, func(t *testing.T) {
			t.Parallel()
			method := authmethod.PrivateKeyFile(filepath.Join("testdata", file), func(string) []byte { return []byte("password") })
			if method == nil {
				t.Error("expected an auth method")
			}
		})
	}
}

func TestPrivateKeyFileBadPassword(t *testing.T) {
	t.Parallel()
	passwords := [][]byte{
		[]byte("secret"),
		[]byte("trustno1"),
		[]byte("password"),
	}

	var attempts = 0
	method := authmethod.PrivateKeyFile(filepath.Join("testdata", "testpass"), func(string) []byte {
		password := passwords[attempts]
		attempts++
		return password
	})

	if method == nil {
		t.Error("expected an auth method")
	}

	if attempts != 3 {
		t.Errorf("expected it to succeed on attempt 3, not %d", attempts)
	}
}
