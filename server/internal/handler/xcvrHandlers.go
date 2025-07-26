package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"rvcx/internal/atputils"
	"rvcx/internal/db"
	"rvcx/internal/lex"
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
	var pu db.ProfileUpdate
	pu.DID = did
	if p.DisplayName != nil {
		if atputils.ValidateGraphemesAndLength(*p.DisplayName, 64, 640) {
			h.badRequest(w, errors.New("displayname too long"))
			return
		}
		pu.Name = p.DisplayName
		pu.UpdateName = true
	}
	if p.DefaultNick != nil {
		if atputils.ValidateLength(*p.DefaultNick, 16) {
			h.badRequest(w, errors.New("nick too long"))
		}
		pu.Nick = p.DefaultNick
		pu.UpdateNick = true
	}
	if p.Status != nil {
		if atputils.ValidateGraphemesAndLength(*p.Status, 640, 6400) {
			h.badRequest(w, errors.New("status too long"))
		}
		pu.Status = p.Status
		pu.UpdateStatus = true
	}
	if p.Avatar != nil {
		// TODO think about how to do avatars!
		pu.Avatar = p.Avatar
		pu.UpdateAvatar = true
	}
	if p.Color != nil {
		if *p.Color > 16777215 || *p.Color < 0 {
			h.badRequest(w, errors.New("color out of bounds"))
		}
		pu.Color = p.Color
		pu.UpdateColor = true
	}
	session, _ := h.sessionStore.Get(r, "oauthsession")
	_, ok := session.Values["id"].(int)
	if !ok {
		h.badRequest(w, errors.New("cannot update profile, not authenticated"))
		return
	}
	profilerecord := lex.ProfileRecord{
		DisplayName: p.DisplayName,
		DefaultNick: p.DefaultNick,
		Status:      p.Status,
		Color:       p.Color,
	}
	client, err := h.getClient(r)
	if err != nil {
		h.serverError(w, err)
		return
	}
	_, err = client.UpdateXCVRProfile(profilerecord, r.Context())
	if err != nil {
		h.serverError(w, err)
		return
	}

	err = h.db.UpdateProfile(pu, r.Context())
	if err != nil {
		h.serverError(w, errors.New("error updating profile: "+err.Error()))
		return
	}

	h.serveProfileView(did, handle, w, r)
}

func (h *Handler) beep(w http.ResponseWriter, r *http.Request) {
	client, err := h.getClient(r)

	if err != nil {
		h.serverError(w, errors.New("error finding client: "+err.Error()))
	}
	client.MakeBskyPost("beep_", r.Context())
}
