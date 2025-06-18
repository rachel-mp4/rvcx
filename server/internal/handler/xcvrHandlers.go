package handler

import (
	"net/http"
	"xcvr-backend/internal/types"
	"xcvr-backend/internal/db"
	"errors"
	"encoding/json"
)

func (h *Handler) postProfile(w http.ResponseWriter, r *http.Request) {
	session, _ := h.sessionStore.Get(r, "oauthsession")
	did, ok := session.Values["did"].(string)
	if !ok || did == "" {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
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
	h.db.UpdateProfile(pu, r.Context())
}
