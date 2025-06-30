package main

import (
	cbg "github.com/whyrusleeping/cbor-gen"
	"xcvr-backend/internal/lex"
)

func main() {
	if err := cbg.WriteMapEncodersToFile("internal/lex/lexicons_cbor.go", "lex",
		lex.ProfileRecord{},
		lex.ChannelRecord{},
		lex.MessageRecord{},
		lex.SignetRecord{}); err != nil {
		panic(err)
	}
}
