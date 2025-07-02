package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/http"
	"time"
	"xcvr-backend/internal/atputils"
	"xcvr-backend/internal/lex"
	"xcvr-backend/internal/model"
	"xcvr-backend/internal/types"
)

func (h *Handler) acceptWebsocket(w http.ResponseWriter, r *http.Request) {
	rkey := r.PathValue("rkey")
	user := r.PathValue("user")
	uri := fmt.Sprintf("at://%s/org.xcvr.feed.channel/%s", user, rkey)
	f, err := model.GetWSHandlerFrom(uri)
	if err != nil {
		http.NotFound(w, r)
		h.logger.Deprintf("couldn't find user %s's server %s", user, rkey)
		h.logger.Println(err.Error())
		return
	}
	f(w, r)
}

func (h *Handler) postChannel(w http.ResponseWriter, r *http.Request) {
	session, _ := h.sessionStore.Get(r, "oauthsession")
	_, ok := session.Values["id"].(uint)
	if !ok {
		h.postMyChannel(w, r)
		return
	}
	client, err := h.getClient(r)
	if err != nil {
		h.serverError(w, errors.New("couldn't find client: "+err.Error()))
		return
	}

	lcr, now, err := h.parseChannelRequest(r)
	if err != nil {
		h.badRequest(w, err)
		return
	}
	uri, cid, err := client.CreateXCVRChannel(lcr, r.Context())
	if err != nil {
		h.serverError(w, errors.New("something bad probs happened when posting a channel "+err.Error()))
		return
	}
	channel := types.Channel{
		URI:       uri,
		CID:       cid,
		DID:       session.Values["did"].(string),
		Host:      lcr.Host,
		Title:     lcr.Title,
		Topic:     lcr.Topic,
		CreatedAt: *now,
		IndexedAt: time.Now(),
	}
	err = h.db.StoreChannel(channel, r.Context())
	if err != nil {
		h.serverError(w, errors.New("well... the record posted but i couldn't store it: "+err.Error()))
		return
	}
	h.getChannels(w, r)
}

func (h *Handler) parseChannelRequest(r *http.Request) (*lex.ChannelRecord, *time.Time, error) {
	var cr types.PostChannelRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&cr)
	if err != nil {
		return nil, nil, errors.New("i think they messed up: " + err.Error())
	}

	var lcr lex.ChannelRecord
	if cr.Title == "" || atputils.ValidateGraphemesAndLength(cr.Title, 64, 640) {
		return nil, nil, errors.New("title empty or too long")
	}
	lcr.Title = cr.Title
	if cr.Host == "" {
		return nil, nil, errors.New("no host")
	}
	lcr.Host = cr.Host
	if cr.Topic != nil {
		if atputils.ValidateGraphemesAndLength(*cr.Topic, 256, 2560) {
			return nil, nil, errors.New("topic too long")
		}
		lcr.Topic = cr.Topic
	}

	dtn := syntax.DatetimeNow()
	lcr.CreatedAt = dtn.String()
	time := dtn.Time()
	return &lcr, &time, nil
}

func (h *Handler) postMyChannel(w http.ResponseWriter, r *http.Request) {
	lcr, now, err := h.parseChannelRequest(r)
	if err != nil {
		h.badRequest(w, err)
		return
	}
	cid, uri, err := h.myClient.CreateXCVRChannel(lcr, r.Context())
	if err != nil {
		h.serverError(w, err)
		return
	}
	mydid, err := atputils.GetMyDid(r.Context())
	if err != nil {
		h.serverError(w, err)
		return
	}
	channel := types.Channel{
		URI:       uri,
		CID:       cid,
		DID:       mydid,
		Host:      lcr.Host,
		Title:     lcr.Title,
		Topic:     lcr.Topic,
		CreatedAt: *now,
		IndexedAt: time.Now(),
	}
	err = h.db.StoreChannel(channel, r.Context())
	if err != nil {
		h.serverError(w, errors.New("sooo... the record posted but i couldn't store it: "+err.Error()))
		return
	}
	h.getChannels(w, r)

}

func (h *Handler) postMessage(w http.ResponseWriter, r *http.Request) {

}
