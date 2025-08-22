package handler

import (
	"encoding/json"
	"errors"
	atoauth "github.com/bluesky-social/indigo/atproto/auth/oauth"
	"net/http"
	"rvcx/internal/types"
)

func (h *Handler) postProfile(cs *atoauth.ClientSession, w http.ResponseWriter, r *http.Request) {
	if cs == nil {
		h.badRequest(w, errors.New("must be logged in!"))
		return
	}
	var p types.PostProfileRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&p)
	if err != nil {
		h.badRequest(w, errors.New("error decoding post profile request: "+err.Error()))
		return
	}
	err = h.rm.PostProfile(cs, r.Context(), &p)
	if err != nil {
		h.serverError(w, errors.New("erroring in postprofile flow: "+err.Error()))
	}
	did := cs.Data.AccountDID.String()
	handle, err := h.db.FullResolveDid(did, r.Context())
	if err != nil {
		h.serverError(w, errors.New("error couldn't resolve did? "+err.Error()))
		return
	}
	h.serveProfileView(did, handle, w, r)
}

func (h *Handler) beep(cs *atoauth.ClientSession, w http.ResponseWriter, r *http.Request) {
	if cs == nil {
		h.badRequest(w, errors.New("must be logged in!"))
		return
	}
	err := h.rm.Beep(cs, r.Context())
	if err != nil {
		h.badRequest(w, err)
		return
	}
	w.Write(nil)
}

func (h *Handler) unfollow(cs *atoauth.ClientSession, w http.ResponseWriter, r *http.Request) {
	if cs == nil {
		h.badRequest(w, errors.New("must be logged in!"))
		return
	}
	qp := r.URL.Query()
	furi := qp.Get("followUri")
	if furi == "" {
		h.badRequest(w, errors.New("must unfollow a user"))
		return
	}
	err := h.rm.Unfollow(cs, furi, r.Context())
	if err != nil {
		h.serverError(w, errors.New("failed to unfollow: "+err.Error()))
	}
	w.Write(nil)
}

func (h *Handler) follow(cs *atoauth.ClientSession, w http.ResponseWriter, r *http.Request) {
	if cs == nil {
		h.badRequest(w, errors.New("must be logged in!"))
		return
	}
	qp := r.URL.Query()
	did := qp.Get("did")
	if did == "" {
		h.badRequest(w, errors.New("must unfollow a user"))
		return
	}
	err := h.rm.Follow(cs, did, r.Context())
	if err != nil {
		h.serverError(w, errors.New("failed to unfollow: "+err.Error()))
	}
	w.Write(nil)
}
