package oauth

import (
	"os"
	"github.com/haileyok/atproto-oauth-golang/helpers"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

var (
	key *jwk.Key
)

func GetJWKS() (*jwk.Key, error) {
	if key != nil {
		return key, nil
	}
	b, err := os.ReadFile("../jwks.json")
	if err != nil {
		return nil, err
	}
	k, err := helpers.ParseJWKFromBytes(b)
	if err != nil {
		return nil, err
	}
	key = &k
	return key, nil
}
