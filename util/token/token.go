package token

import (
	"crypto/rand"

	"github.com/jbenet/go-base58"
)

// TODO: use libsqsc token.NewBase58 instead
func GenSecure(size int) string {
	k := make([]byte, size)
	for bytes := 0; bytes < len(k); {
		n, err := rand.Read(k[bytes:])
		if err != nil {
			panic("rand.Read() failed")
		}
		bytes += n
	}
	return base58.Encode(k)
}
