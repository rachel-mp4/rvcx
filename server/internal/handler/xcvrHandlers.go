package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"rvcx/internal/types"
)

func (h *Handler) postProfile(w http.ResponseWriter, r *http.Request) {
	did, handle, err := h.findDidAndHandle(r)
	if err != nil {
		h.handleFindDidAndHandleError(w, err)
		return
	}
	var p types.PostProfileRequest
	decoder := json.NewDecoder(r.Body)
	err = decoder.Decode(&p)
	if err != nil {
		h.badRequest(w, errors.New("error decoding post profile request: "+err.Error()))
		return
	}
	s, _ := h.sessionStore.Get(r, "oauthsession")
	id, ok := s.Values["id"].(int)
	if !ok {
		h.badRequest(w, errors.New("must be logged in!"))
		return
	}
	err = h.rm.PostProfile(did, id, r.Context(), &p)
	if err != nil {
		h.serverError(w, errors.New("erroring in postprofile flow: "+err.Error()))
	}
	h.serveProfileView(did, handle, w, r)
}

func (h *Handler) beep(w http.ResponseWriter, r *http.Request) {
	s, _ := h.sessionStore.Get(r, "oauthsession")
	id, ok := s.Values["id"].(int)
	if !ok {
		h.badRequest(w, errors.New("must be logged in!"))
		return
	}
	err := h.rm.Beep(id, r.Context())
	if err != nil {
		h.badRequest(w, err)
		return
	}
	w.Write(nil)
}
