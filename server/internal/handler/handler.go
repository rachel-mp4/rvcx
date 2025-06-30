package handler

import (
	"context"
	"github.com/gorilla/sessions"
	"net/http"

	"os"
	"xcvr-backend/internal/atputils"
	"xcvr-backend/internal/db"
	"xcvr-backend/internal/lex"
	"xcvr-backend/internal/log"
	"xcvr-backend/internal/oauth"
)

type Handler struct {
	db           *db.Store
	sessionStore *sessions.CookieStore
	router       *http.ServeMux
	logger       *log.Logger
	oauth        *oauth.Service
	xrpc         *oauth.Client
}

func New(db *db.Store, logger *log.Logger, oauthserv *oauth.Service) *Handler {
	mux := http.NewServeMux()
	sessionStore := sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
	did, err := atputils.GetDidFromHandle(context.Background(), os.Getenv("MY_IDENTITY"))
	if err != nil {
		panic(err)
	}
	pdshost, err := atputils.GetPDSFromDid(context.Background(), did, http.DefaultClient)
	if err != nil {
		panic(err)
	}
	xrpc := oauth.NewXRPCClient(db, logger, pdshost, did)
	err = xrpc.CreateSession(context.Background())
	if err != nil {
		panic(err)
	}
	err = xrpc.CreateXCVRSignet(lex.SignetRecord{
		ChannelURI: "beep.boop",
		LRCID:      11,
		Author:     "sneep.snirp",
	}, context.Background())
	if err != nil {
		panic(err)
	}
	h := &Handler{db, sessionStore, mux, logger, oauthserv, xrpc}
	// lrc handlers
	mux.HandleFunc("GET /lrc/{user}/{rkey}/ws", h.acceptWebsocket)
	mux.HandleFunc("POST /lrc/channel", h.postChannel)
	mux.HandleFunc("POST /lrc/message", h.postMessage)
	// beep handlers
	mux.HandleFunc("POST /xcvr/profile", h.postProfile)
	mux.HandleFunc("POST /xcvr/beep", h.beep)
	// lexicon handlers
	mux.HandleFunc("GET /xrpc/org.xcvr.feed.getChannels", h.getChannels)
	mux.HandleFunc("GET /xrpc/org.xcvr.lrc.getMessages", h.getMessages)
	mux.HandleFunc("GET /xrpc/org.xcvr.actor.resolveChannel", h.resolveChannel)
	mux.HandleFunc("GET /xrpc/org.xcvr.actor.getProfileView", h.getProfileView)
	// backend metadata handlers
	mux.HandleFunc(clientMetadataPath(), h.serveClientMetadata)
	mux.HandleFunc(clientTOSPath(), h.serveTOS)
	mux.HandleFunc(clientPolicyPath(), h.servePolicy)
	// oauth handlers
	mux.HandleFunc(oauthJWKSPath(), h.serveJWKS)
	mux.HandleFunc("POST /oauth/login", h.oauthLogin)
	mux.HandleFunc("GET /oauth/whoami", h.getSession)
	mux.HandleFunc(oauthCallbackPath(), h.oauthCallback)
	return h
}

func (h *Handler) badRequest(w http.ResponseWriter, err error) {
	h.logger.Deprintln(err.Error())
	w.Header().Set("Content-Type", "application/json")
	http.Error(w, `{"error":"Invalid JSON","message":"Could not parse request body"}`, http.StatusBadRequest)
}

func (h *Handler) serverError(w http.ResponseWriter, err error) {
	h.logger.Println(err.Error())
	w.Header().Set("Content-Type", "application/json")
	http.Error(w, `{"error":"Internal server error","message":"Something went wrong"}`, http.StatusInternalServerError)
}

func (h *Handler) notFound(w http.ResponseWriter, err error) {
	h.logger.Println(err.Error())
	w.Header().Set("Content-Type", "application/json")
	http.Error(w, `{"error":"Not Found","message":"I couldn't find your resource"}`, http.StatusNotFound)
}

func (h *Handler) WithCORSAll() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.logger.Deprintf("incoming request: %s %s", r.Method, r.URL.Path)
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.router.ServeHTTP(w, r)
	})
}
