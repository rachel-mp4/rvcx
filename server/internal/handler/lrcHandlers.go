package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/http"
	"os"
	"time"
	"xcvr-backend/internal/atputils"
	"xcvr-backend/internal/lex"
	"xcvr-backend/internal/types"
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
	err = h.db.StoreChannel(&channel, r.Context())
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
	channel := types.Channel{
		URI:       uri,
		CID:       cid,
		DID:       atputils.GetMyDid(),
		Host:      lcr.Host,
		Title:     lcr.Title,
		Topic:     lcr.Topic,
		CreatedAt: *now,
		IndexedAt: time.Now(),
	}
	err = h.db.StoreChannel(&channel, r.Context())
	if err != nil {
		h.serverError(w, errors.New("sooo... the record posted but i couldn't store it: "+err.Error()))
		return
	}
	h.getChannels(w, r)

}

func (h *Handler) parseMessageRequest(r *http.Request) (*lex.MessageRecord, *time.Time, error) {
	var mr types.PostMessageRequest
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&mr)
	if err != nil {
		return nil, nil, errors.New("couldn't decode: " + err.Error())
	}
	if mr.SignetURI == nil {
		if mr.MessageID == nil || mr.ChannelURI == nil {
			return nil, nil, errors.New("must provide a way to determine signet")
		}
		signetUri, err := h.db.QuerySignet(*mr.ChannelURI, *mr.MessageID, r.Context())
		if err != nil {
			return nil, nil, errors.New("i couldn't find the signet :c : " + err.Error())
		}
		mr.SignetURI = &signetUri
	}
	var lmr lex.MessageRecord
	lmr.SignetURI = *mr.SignetURI
	lmr.Body = mr.Body
	if mr.Nick != nil {
		nick := *mr.Nick
		if atputils.ValidateLength(nick, 16) {
			return nil, nil, errors.New("that nick is too long")
		}
	}
	lmr.Nick = mr.Nick

	if mr.Color != nil {
		color := uint64(*mr.Color)
		if color > 16777215 {
			return nil, nil, errors.New("that color is too big")
		}
	}
	now := syntax.DatetimeNow()
	lmr.PostedAt = now.String()
	nt := now.Time()
	return &lmr, &nt, nil
}

func (h *Handler) postMyMessage(w http.ResponseWriter, r *http.Request) {

}

func (h *Handler) postMessage(w http.ResponseWriter, r *http.Request) {
	session, _ := h.sessionStore.Get(r, "oauthsession")
	_, ok := session.Values["id"].(uint)
	if !ok {
		h.postMyMessage(w, r)
		return
	}
	client, err := h.getClient(r)
	if err != nil {
		h.serverError(w, errors.New("couldn't find client: "+err.Error()))
		return
	}

	lmr, now, err := h.parseMessageRequest(r)
	if err != nil {
		h.badRequest(w, errors.New("couldn't parse message "+err.Error()))
		return
	}

	uri, cid, err := client.CreateXCVRMessage(*lmr, r.Context())
	if err != nil {
		h.serverError(w, errors.New("couldn't add to user repo: "))
		return
	}
	did := session.Values["did"].(string)
	var coloruint32ptr *uint32
	if lmr.Color != nil {
		color := uint32(*lmr.Color)
		coloruint32ptr = &color
	}
	message := types.Message{
		URI:       uri,
		DID:       did,
		CID:       cid,
		SignetURI: lmr.SignetURI,
		Body:      lmr.Body,
		Nick:      lmr.Nick,
		Color:     coloruint32ptr,
		PostedAt:  *now,
	}
	err = h.db.StoreMessage(&message, r.Context())
	if err != nil {
		h.serverError(w, errors.New("sooo... the record posted but i couldn't store it: "+err.Error()))
		return
	}
	h.getMessages(w, r)
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
