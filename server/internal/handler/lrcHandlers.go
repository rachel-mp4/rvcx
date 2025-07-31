package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"rvcx/internal/atputils"
	"rvcx/internal/types"
)

func (h *Handler) acceptWebsocket(w http.ResponseWriter, r *http.Request) {
	rkey := r.PathValue("rkey")
	user := r.PathValue("user")
	uri := fmt.Sprintf("at://%s/org.xcvr.feed.channel/%s", user, rkey)
	f, err := h.model.GetWSHandlerFrom(uri)
	if err != nil {
		http.NotFound(w, r)
		h.logger.Deprintf("couldn't find user %s's server %s", user, rkey)
		h.logger.Println(err.Error())
		return
	}
	f(w, r)
}

func (h *Handler) postChannel(w http.ResponseWriter, r *http.Request) {
	cr, err := h.parseChannelRequest(r)
	if err != nil {
		h.badRequest(w, err)
		return
	}
	session, _ := h.sessionStore.Get(r, "oauthsession")
	id, ok := session.Values["id"].(int)
	var uri, did string
	if !ok {
		did, uri, err = h.rm.PostMyChannel(r.Context(), cr)
	} else {
		udid, ok := session.Values["did"].(string)
		if !ok {
			h.badRequest(w, errors.New("user session has no did?"))
			return
		}
		did, uri, err = h.rm.PostChannel(id, udid, r.Context(), cr)
	}
	if err != nil {
		h.serverError(w, err)
		return
	}
	handle, err := h.db.ResolveDid(did, r.Context())
	if err != nil {
		handle, err = atputils.TryLookupDid(r.Context(), did)
		if err != nil {
			h.serverError(w, errors.New(fmt.Sprintf("couldn't find handle for did %s: %s", did, err.Error())))
			return
		}
		go h.db.StoreDidHandle(did, handle, context.Background())
	}
	rkey, _ := atputils.RkeyFromUri(uri)
	http.Redirect(w, r, fmt.Sprintf("/c/%s/%s", handle, rkey), http.StatusSeeOther)
}

func (h *Handler) parseChannelRequest(r *http.Request) (*types.PostChannelRequest, error) {
	var cr types.PostChannelRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&cr)
	if err != nil {
		return nil, errors.New("i think they messed up: " + err.Error())
	}
	return &cr, nil
}

func (h *Handler) parseMessageRequest(r *http.Request) (*types.PostMessageRequest, error) {
	var mr types.PostMessageRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&mr)
	if err != nil {
		return nil, errors.New("couldn't decode: " + err.Error())
	}
	return &mr, nil
}

func (h *Handler) postMessage(w http.ResponseWriter, r *http.Request) {
	pmr, err := h.parseMessageRequest(r)
	if err != nil {
		h.badRequest(w, errors.New("failed to parse message request: "+err.Error()))
		return
	}
	session, _ := h.sessionStore.Get(r, "oauthsession")
	id, ok := session.Values["id"].(int)
	if !ok {
		err = h.rm.PostMyMessage(r.Context(), pmr)
	} else {
		did, ok := session.Values["did"].(string)
		if !ok {
			h.badRequest(w, errors.New("has sid but doesn't have sdid!"))
		}
		err = h.rm.PostMessage(id, did, r.Context(), pmr)
	}
	if err != nil {
		h.serverError(w, errors.New("error posting message: "+err.Error()))
	}
	w.Write(nil)
}

func (h *Handler) deleteChannel(w http.ResponseWriter, r *http.Request) {
	did, handle, err := h.findDidAndHandle(r)
	if err != nil {
		h.logger.Deprintln("tried to anonymously delete")
		return
	}
	rkey := r.PathValue("rkey")
	user := r.PathValue("user")
	if did != user && handle != os.Getenv("ADMIN_HANDLE") {
		h.logger.Deprintln("tried to delete not logged in")
		return
	}
	uri := fmt.Sprintf("at://%s/org.xcvr.feed.channel/%s", user, rkey)
	err = h.db.DeleteChannel(uri, r.Context())
	if err != nil {
		h.logger.Deprintln("failed to delete")
		return
	}
	h.getChannels(w, r)
}

func (h *Handler) subscribeLexStream(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	f, err := h.model.GetLexStreamFrom(uri)
	if err != nil {
		http.NotFound(w, r)
		h.logger.Deprintf("couldn't find server %s", uri)
		h.logger.Println(err.Error())
		return
	}
	f(w, r)
}
