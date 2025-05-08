package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"xcvr-backend/internal/db"
	"xcvr-backend/internal/log"
	"xcvr-backend/internal/model"

	"github.com/jackc/pgx/v5"
)

type Handler struct {
	db     *pgx.Conn
	router *http.ServeMux
	logger log.Logger
}

func New(conn *pgx.Conn, logger log.Logger) *Handler {
	mux := http.NewServeMux()
	h := &Handler{conn, mux, logger}
	mux.HandleFunc("GET /lrc/{title}/ws", h.acceptWebsocket)
	mux.HandleFunc("GET /lrc/{user}/{title}/ws", h.acceptWebsocketUser)
	mux.HandleFunc("GET /xrpc/org.xcvr.feed.getChannels", h.getChannels)
	mux.HandleFunc("GET /xrpc/org.xcvr.lrc.getMessages", h.getMessages)
	mux.HandleFunc("POST /lrc/channel", postChannel)
	mux.HandleFunc("POST /lrc/message", postMessage)
	return h
}

func (h *Handler) acceptWebsocket(w http.ResponseWriter, r *http.Request) {
	title := r.PathValue("title")
	f, err := model.GetWSHandlerFrom(title, h.db)
	if err != nil {
		http.NotFound(w, r)
		h.logger.Deprintf("couldn't find server %s", title)
		return
	}
	f(w, r)
}

func (h *Handler) acceptWebsocketUser(w http.ResponseWriter, r *http.Request) {
	title := r.PathValue("title")
	user := r.PathValue("user")
	f, err := model.GetWSHandlerFrom(title, h.db)
	if err != nil {
		http.NotFound(w, r)
		h.logger.Deprintf("couldn't find user %s's server %s", user, title)
		return
	}
	f(w, r)
}


func (h *Handler) getMessages(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) getChannels(w http.ResponseWriter, r *http.Request) {
	limitstr := r.URL.Query().Get("limit")
	limit := 50
	if limitstr != "" {
		l, err := strconv.Atoi(limitstr)
		if err == nil {
			limit = max(min(l, 100),1)
		}
	}
	cvs, err := db.GetChannelViews(limit, r.Context(), h.db)
	if err != nil {
		serverError(w)
		h.logger.Printf("db.GetChannels failed! %s", err.Error())
		return
	}
	encoder := json.NewEncoder(w)
	encoder.Encode(cvs)
}

func postChannel(w http.ResponseWriter, r *http.Request) {

}

func postMessage(w http.ResponseWriter, r *http.Request) {

}

func badRequest(w http.ResponseWriter) {
	http.Error(w, `{"error":"Invalid JSON","message":"Could not parse request body"}`,http.StatusBadRequest)
}

func serverError(w http.ResponseWriter) {
	http.Error(w, `{"error":"Internal server error","message":"Something went wrong"}`,http.StatusInternalServerError)
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