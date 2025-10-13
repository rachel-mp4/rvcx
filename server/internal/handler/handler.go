package handler

import (
	"github.com/gorilla/sessions"
	"net/http"

	"os"
	"rvcx/internal/db"
	"rvcx/internal/log"
	"rvcx/internal/model"
	"rvcx/internal/oauth"
	"rvcx/internal/recordmanager"
)

type Handler struct {
	db           *db.Store
	sessionStore *sessions.CookieStore
	router       *http.ServeMux
	logger       *log.Logger
	oauth        *oauth.Service
	model        *model.Model
	rm           *recordmanager.RecordManager
}

func New(db *db.Store, logger *log.Logger, oauthserv *oauth.Service, model *model.Model, recordmanager *recordmanager.RecordManager) *Handler {
	mux := http.NewServeMux()
	sessionStore := sessions.NewCookieStore([]byte(os.Getenv("SESSION_KEY")))
	h := &Handler{db, sessionStore, mux, logger, oauthserv, model, recordmanager}
	// lrc handlers
	mux.HandleFunc("GET /lrc/{user}/{rkey}/ws", h.WithCORS(h.acceptWebsocket))
	mux.HandleFunc("DELETE /lrc/{user}/{rkey}/ws", h.oauthMiddleware(h.deleteChannel))
	mux.HandleFunc("POST /lrc/channel", h.oauthMiddleware(h.postChannel))
	mux.HandleFunc("POST /lrc/message", h.oauthMiddleware(h.postMessage))
	mux.HandleFunc("POST /lrc/image", h.oauthMiddleware(h.uploadImage))
	mux.HandleFunc("POST /lrc/media", h.oauthMiddleware(h.postMedia))
	mux.HandleFunc("GET  /lrc/image", h.WithCORS(h.getImage))
	mux.HandleFunc("POST /lrc/mymessage", h.postMyMessage)
	// xcvr handlers
	mux.HandleFunc("POST /xcvr/profile", h.oauthMiddleware(h.postProfile))
	mux.HandleFunc("POST /xcvr/beep", h.oauthMiddleware(h.beep))
	// lexicon handlers
	mux.HandleFunc("GET /xrpc/org.xcvr.feed.getChannels", h.WithCORS(h.getChannels))
	mux.HandleFunc("GET /xrpc/org.xcvr.feed.getChannel", h.WithCORS(h.getChannel))
	mux.HandleFunc("GET /xrpc/org.xcvr.lrc.getMessages", h.WithCORS(h.getMessages))
	mux.HandleFunc("GET /xrpc/org.xcvr.lrc.getImage", h.WithCORS(h.getImage))
	mux.HandleFunc("GET /xrpc/org.xcvr.actor.resolveChannel", h.WithCORS(h.resolveChannel))
	mux.HandleFunc("GET /xrpc/org.xcvr.actor.getProfileView", h.WithCORS(h.getProfileView))
	mux.HandleFunc("GET /xrpc/org.xcvr.lrc.subscribeLexStream", h.WithCORS(h.subscribeLexStream))
	mux.HandleFunc("GET /xrpc/org.xcvr.actor.getLastSeen", h.WithCORS(h.getLastSeen))
	// backend metadata handlers
	mux.HandleFunc(clientMetadataPath(), h.WithCORS(h.serveClientMetadata))
	mux.HandleFunc(clientTOSPath(), h.WithCORS(h.serveTOS))
	mux.HandleFunc(clientPolicyPath(), h.WithCORS(h.servePolicy))
	// oauth handlers
	mux.HandleFunc(oauthJWKSPath(), h.WithCORS(h.serveJWKS))
	mux.HandleFunc("POST /oauth/login", h.oauthLogin)
	mux.HandleFunc("POST /oauth/logout", h.oauthMiddleware(h.oauthLogout))
	mux.HandleFunc("POST /oauth/ban", h.postBan)
	mux.HandleFunc("GET /oauth/ban", h.getBan)
	mux.HandleFunc("GET /oauth/whoami", h.getSession)
	mux.HandleFunc(oauthCallbackPath(), h.WithCORS(h.oauthCallback))
	return h
}

func (h *Handler) badRequest(w http.ResponseWriter, err error) {
	h.logger.Deprintln(err.Error())
	w.Header().Set("Content-Type", "application/json")
	http.Error(w, `{
		"error":"Invalid JSON", 
		"message":"Could not parse request body"
	}`, http.StatusBadRequest)
}

func (h *Handler) serverError(w http.ResponseWriter, err error) {
	h.logger.Println(err.Error())
	w.Header().Set("Content-Type", "application/json")
	http.Error(w, `{"error":"Internal server error", "message":"Something went wrong"}`, http.StatusInternalServerError)
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

func (h *Handler) WithCORS(f func(w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorizaton, X-Requested-With, Sec-WebSocket-Protocol, Sec-WebSocket-Extensions, Sec-WebSocket-Key, Sec-WebSocket-Version")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
		if r.Method == "Options" {
			w.WriteHeader(http.StatusOK)
			return
		}
		f(w, r)
	}
}

func (h *Handler) Serve() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h.logger.Deprintf("incoming request: %s %s", r.Method, r.URL.Path)
		h.router.ServeHTTP(w, r)
	})
}
