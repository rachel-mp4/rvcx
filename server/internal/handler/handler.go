package handler

import (
	"github.com/gorilla/sessions"
	"net/http"

	"os"
	"xcvr-backend/internal/db"
	"xcvr-backend/internal/log"
	"xcvr-backend/internal/model"
	"xcvr-backend/internal/oauth"
)

type Handler struct {
	db           *db.Store
	sessionStore *sessions.CookieStore
	router       *http.ServeMux
	logger       *log.Logger
	oauth        *oauth.Service
	myClient     *oauth.PasswordClient
	clientmap    *oauth.ClientMap
	model        *model.Model
}

func New(db *db.Store, logger *log.Logger, oauthserv *oauth.Service, xrpc *oauth.PasswordClient, model *model.Model) *Handler {
	mux := http.NewServeMux()
	sessionStore := sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
	clientmap := oauth.NewClientMap()
	h := &Handler{db, sessionStore, mux, logger, oauthserv, xrpc, clientmap, model}
	// lrc handlers
	mux.HandleFunc("GET /lrc/{user}/{rkey}/ws", h.acceptWebsocket)
	mux.HandleFunc("DELETE /lrc/{user}/{rkey}/ws", h.deleteChannel)
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
	mux.HandleFunc("GET /xrpc/org.xcvr.lrc.getLexStream", h.subscribeLexStream)
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
