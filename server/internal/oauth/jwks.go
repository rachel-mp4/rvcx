package oauth

import (
	"github.com/bluesky-social/indigo/atproto/crypto"
	"os"
)

func GetPrivateKey() (*crypto.PrivateKeyK256, error) {
	csk := os.Getenv("CLIENT_SECRET_KEY")
	key, err := crypto.ParsePrivateBytesK256([]byte(csk))
	if err != nil {
		return nil, err
	}
	return key, nil
}
