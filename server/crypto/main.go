package main

import (
	"fmt"
	"github.com/bluesky-social/indigo/atproto/crypto"
)

func main() {
	privateKey, err := crypto.GeneratePrivateKeyK256()
	if err != nil {
		panic(err)
	}
	clientSecretKey := privateKey.Multibase()
	fmt.Println(clientSecretKey)
}
