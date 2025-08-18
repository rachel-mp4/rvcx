package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"rvcx/internal/atplistener"
	"rvcx/internal/atputils"
	"rvcx/internal/db"
	"rvcx/internal/handler"
	"rvcx/internal/log"
	"rvcx/internal/model"
	"rvcx/internal/oauth"
	"rvcx/internal/recordmanager"

	"github.com/joho/godotenv"
)

func main() {
	logger := log.New(os.Stdout, true)

	gdeerr := godotenv.Load("../.env")
	if gdeerr != nil {
		logger.Println("i think you should make a .env file in the rvcx directory !\n\nExample contents:\n-------------------------------------------------------------------\nPOSTGRES_USER=xcvr\nPOSTGRES_PASSWORD=secret\nPOSTGRES_DB=xcvrdb\nPOSTGRES_PORT=15432\n-------------------------------------------------------------------\n\nGood luck !\n\n")
		panic(gdeerr)
	}
	store, err := db.Init()
	defer store.Close()
	if err != nil {
		logger.Println("failed to init db")
		panic(err)
	}
	host, err := atputils.GetPDSFromHandle(context.Background(), atputils.GetMyHandle())
	if err != nil {
		panic(err)
	}
	did := atputils.GetMyDid()
	if did == "" {
		panic(errors.New("WOOPS I MESSED UP THE DID"))
	}
	xrpc := oauth.NewPasswordClient(did, host, logger)
	err = xrpc.CreateSession(context.Background())
	if err != nil {
		panic(err)
	}
	oauthclient, err := oauth.NewService(*store)
	if err != nil {
		logger.Println(err.Error())
		panic(err)
	}
	recordmanager := recordmanager.New(logger, store, xrpc, oauthclient)
	model := model.Init(store, logger, xrpc, recordmanager)
	recordmanager.SetBroadcaster(model)
	h := handler.New(store, logger, oauthclient, model, recordmanager)
	go consumeLoop(context.Background(), store, logger, xrpc, recordmanager)
	http.ListenAndServe(":8080", h.Serve())

}

const (
	defaultServerAddr = "wss://jetstream.atproto.tools/subscribe"
)

func consumeLoop(ctx context.Context, db *db.Store, l *log.Logger, cli *oauth.PasswordClient, rm *recordmanager.RecordManager) {
	jsServerAddr := os.Getenv("JS_SERVER_ADDR")
	if jsServerAddr == "" {
		jsServerAddr = defaultServerAddr
	}
	consumer := atplistener.NewConsumer(jsServerAddr, l, db, cli, rm)
	for {
		err := consumer.Consume(ctx)
		if err != nil {
			l.Deprintf("error in consume loop: %s", err.Error())
			if errors.Is(err, context.Canceled) {
				l.Deprintf("exiting consume loop")
				return
			}
		}
	}
}
