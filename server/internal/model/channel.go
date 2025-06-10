package model

import (
	"context"
	"errors"
	"net/http"
	"os"
	"xcvr-backend/internal/db"

	"github.com/rachel-mp4/lrcd"
)

var (
	validServer map[string]bool
	uriToServer = make(map[string]*lrcd.Server)
)

func GetWSHandlerFrom(uri string) (http.HandlerFunc, error) {
	server, err := getServer(uri)
	if err != nil {
		return nil, err
	}
	return server.WSHandler(), nil
}

func Init(store *db.Store) {
	uris, err := store.GetChannelURIs(context.Background())
	if err != nil {
		panic(err)
	}
	validServer = make(map[string]bool, len(uris))
	myid := os.Getenv("MY_IDENTITY")
	for _, uri := range uris {
		validServer[uri.URI] = (uri.Host == myid)
	}
}

func getServer(uri string) (*lrcd.Server, error) {
	if !validServer[uri] {
		return nil, errors.New("Not a valid server")
	}
	server, ok := uriToServer[uri]
	if !ok {
		var err error
		server, err = lrcd.NewServer(lrcd.WithLogging(os.Stdout,true))
		if err != nil {
			return nil, errors.New("Error creating server")
		}
		uriToServer[uri] = server
		err = server.Start()
		if err != nil {
			return nil, errors.New("Error starting server")
		}
	}
	return server, nil
}

