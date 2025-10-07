package main

import (
	cbg "github.com/whyrusleeping/cbor-gen"
	"rvcx/internal/lex"
)

func main() {
	if err := cbg.WriteMapEncodersToFile("internal/lex/lexicons_cbor.go", "lex",
		lex.ProfileRecord{},
		lex.ChannelRecord{},
		lex.MessageRecord{},
		lex.SignetRecord{},
		lex.AspectRatio{},
		lex.Image{},
		lex.MediaRecord{}); err != nil {
		panic(err)
	}
}
