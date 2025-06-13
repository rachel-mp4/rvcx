package handler

import (
	"xcvr-backend/internal/types"
	"errors"
	"strconv"
	"fmt"
	"encoding/json"
	"net/http"
)

func (h *Handler) getChannels(w http.ResponseWriter, r *http.Request) {
	limitstr := r.URL.Query().Get("limit")
	limit := 50
	if limitstr != "" {
		l, err := strconv.Atoi(limitstr)
		if err == nil {
			limit = max(min(l, 100), 1)
		}
	}
	cvs, err := h.db.GetChannelViews(limit, r.Context())
	if err != nil {
		h.serverError(w, err)
		h.logger.Printf("db.GetChannels failed! %s", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(cvs)
}

func (h *Handler) getMessages(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) resolveChannel(w http.ResponseWriter, r *http.Request) {
	handle := r.URL.Query().Get("handle")
	did := r.URL.Query().Get("did")
	rkey := r.URL.Query().Get("rkey")
	if did == "" {
		if handle == "" {
			h.badRequest(w, errors.New("did not provide did or handle"))
			return
		}
		var err error
		did, err = h.db.ResolveHandle(handle, r.Context())
		if err != nil {
			h.serverError(w, err)
			return
		}
	}
	url := fmt.Sprintf("/lrc/%s/%s/ws", did, rkey)
	rchanres := types.ResolveChannelResponse{URL: url}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(rchanres)
}
