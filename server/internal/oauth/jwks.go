package oauth

import (
	"github.com/bluesky-social/indigo/atproto/crypto"
	"os"
)

func GetPrivateKey() (crypto.PrivateKeyExportable, error) {
	csk := os.Getenv("CLIENT_SECRET_KEY")
	key, err := crypto.ParsePrivateMultibase(csk)
	if err != nil {
		return nil, err
	}
	return key, nil
}
