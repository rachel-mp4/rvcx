package handler

import (
	"encoding/json"
	"errors"
	"github.com/rivo/uniseg"
	"net/http"
	"unicode/utf16"
	"xcvr-backend/internal/db"
	"xcvr-backend/internal/types"
)

func (h *Handler) postProfile(w http.ResponseWriter, r *http.Request) {
	did, handle, err := h.findDidAndHandle(w, r)
	if err != nil {
		h.handleFindDidAndHandleError(w, r, err)
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
		if uniseg.GraphemeClusterCount(*p.DisplayName) > 64 {
			h.badRequest(w, errors.New("too many graphemes"))
			return
		}
		runes := []rune(*p.DisplayName)
		us := utf16.Encode(runes)
		if len(us) > 640 {
			h.badRequest(w, errors.New("too many utf16 code points"))
			return
		}
		pu.Name = p.DisplayName
		pu.UpdateName = true
	}
	if p.DefaultNick != nil {
		runes := []rune(*p.DefaultNick)
		us := utf16.Encode(runes)
		if len(us) > 16 {
			h.badRequest(w, errors.New("too many utf16 code points"))
			return
		}
		pu.Nick = p.DefaultNick
		pu.UpdateNick = true
	}
	if p.Status != nil {
		if uniseg.GraphemeClusterCount(*p.DisplayName) > 640 {
			h.badRequest(w, errors.New("too many graphemes"))
			return
		}
		runes := []rune(*p.DisplayName)
		us := utf16.Encode(runes)
		if len(us) > 6400 {
			h.badRequest(w, errors.New("too many utf16 code points"))
			return
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
	err = h.db.UpdateProfile(pu, r.Context())
	if err != nil {
		h.serverError(w, errors.New("error updating profile: "+err.Error()))
		return
	}
	h.serveProfileView(did, handle, w, r)
}
