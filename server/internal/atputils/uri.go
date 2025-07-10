package atputils

import (
	"errors"
	"fmt"
	"strings"
)

func URI(did string, collection string, rkey string) string {
	return fmt.Sprintf("at://%s/%s/%s", did, collection, rkey)
}

func DidFromUri(uri string) (did string, err error) {
	s, err := trimScheme(uri)
	if err != nil {
		return
	}
	ss, err := uriFragSplit(s)
	if err != nil {
		return
	}
	did = ss[0]
	return
}

func trimScheme(uri string) (string, error) {
	s, ok := strings.CutPrefix(uri, "at://")
	if !ok {
		return "", errors.New("not a uri, missing at:// scheme")
	}
	return s, nil
}

func uriFragSplit(urifrag string) ([]string, error) {
	ss := strings.Split(urifrag, "/")
	if len(ss) != 3 {
		return nil, errors.New("not a urifrag, incorrect number of bits")
	}
	return ss, nil
}

func RkeyFromUri(uri string) (rkey string, err error) {
	s, err := trimScheme(uri)
	if err != nil {
		return
	}
	ss, err := uriFragSplit(s)
	if err != nil {
		return
	}
	rkey = ss[2]
	return
}
