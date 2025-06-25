package handler

import (
	"net/http"
	"xcvr-backend/internal/types"
	"xcvr-backend/internal/db"
	"errors"
	"encoding/json"
)

func (h *Handler) postProfile(w http.ResponseWriter, r *http.Request) {
	did, handle, err := h.findDidAndHandle(w, r)
	if err != nil {
		h.handleFindDidAndHandleError(w,r, err)
		return
	}
	var p types.PostProfileRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&p)
	if err != nil {
		h.badRequest(w, errors.New("error decoding post profile request: " + err.Error()))
		return
	}
	var pu db.ProfileUpdate
	pu.DID = did
	if p.DisplayName != nil {
		pu.Name = p.DisplayName
		pu.UpdateName = true
	}	
	if p.DefaultNick != nil {
		pu.Nick = p.DefaultNick
		pu.UpdateNick = true
	}	
	if p.Status != nil {
		pu.Status = p.Status
		pu.UpdateStatus = true
	}	
	if p.Avatar != nil {
		pu.Avatar = p.Avatar
		pu.UpdateAvatar = true
	}	
	if p.Color != nil {
		pu.Color = p.Color
		pu.UpdateColor = true
	}	
	err = h.db.UpdateProfile(pu, r.Context())
	if err != nil {
		h.serverError(w, errors.New("error updating profile: " + err.Error()))
		return
	}
	h.serveProfileView(did, handle, w,r)
}
