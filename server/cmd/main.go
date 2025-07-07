package main

import (
	"context"
	"errors"
	"net/http"
	"os"
	"time"
	"xcvr-backend/internal/atplistener"
	"xcvr-backend/internal/atputils"
	"xcvr-backend/internal/db"
	"xcvr-backend/internal/handler"
	"xcvr-backend/internal/log"
	"xcvr-backend/internal/model"
	"xcvr-backend/internal/oauth"

	"github.com/joho/godotenv"
)

func main() {
	logger := log.New(os.Stdout, true)

	gdeerr := godotenv.Load("../.env")
	if gdeerr != nil {
		logger.Println("i think you should make a .env file in the xcvr-backend directory !\n\nExample contents:\n-------------------------------------------------------------------\nPOSTGRES_USER=xcvr\nPOSTGRES_PASSWORD=secret\nPOSTGRES_DB=xcvrdb\nPOSTGRES_PORT=15432\n-------------------------------------------------------------------\n\nGood luck !\n\n")
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
	model := model.Init(store, logger, xrpc)
	httpclient := &http.Client{
		Timeout: 5 * time.Second,
		Transport: &http.Transport{
			IdleConnTimeout: 90 * time.Second,
		},
	}
	oauthclient, err := oauth.NewService(httpclient)
	if err != nil {
		logger.Println(err.Error())
		panic(err)
	}
	h := handler.New(store, logger, oauthclient, xrpc, model)
	go consumeLoop(context.Background(), store, logger)
	http.ListenAndServe(":8080", h.WithCORSAll())

}

const (
	defaultServerAddr = "wss://jetstream.atproto.tools/subscribe"
)

func consumeLoop(ctx context.Context, db *db.Store, l *log.Logger) {
	jsServerAddr := os.Getenv("JS_SERVER_ADDR")
	if jsServerAddr == "" {
		jsServerAddr = defaultServerAddr
	}
	consumer := atplistener.NewConsumer(jsServerAddr, l, db)
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
