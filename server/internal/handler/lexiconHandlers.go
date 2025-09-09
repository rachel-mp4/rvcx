package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"rvcx/internal/atputils"
	"rvcx/internal/types"
	"strconv"
	"time"
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
func (h *Handler) getChannel(w http.ResponseWriter, r *http.Request) {
	uri := r.URL.Query().Get("uri")
	handle := r.URL.Query().Get("handle")
	rkey := r.URL.Query().Get("rkey")
	var cv *types.ChannelView
	var err error
	if uri != "" {
		cv, err = h.db.GetChannelView(uri, r.Context())
	} else {
		cv, err = h.db.GetChannelViewHR(handle, rkey, r.Context())
	}
	if err != nil {
		h.notFound(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(cv)
}

func (h *Handler) getMessages(w http.ResponseWriter, r *http.Request) {
	limitstr := r.URL.Query().Get("limit")
	limit := 50
	if limitstr != "" {
		l, err := strconv.Atoi(limitstr)
		if err == nil {
			limit = max(min(l, 100), 1)
		}
	}
	cursorstr := r.URL.Query().Get("cursor")
	var cursor *int
	if cursorstr != "" {
		c, err := strconv.Atoi(cursorstr)
		if err == nil {
			cursor = &c
		}
	}
	channelURI := r.URL.Query().Get("channelURI")
	messages, err := h.db.GetMessages(channelURI, limit, cursor, r.Context())
	if err != nil {
		h.serverError(w, errors.New("something went south: "+err.Error()))
		return
	}
	var gmo types.GetMessagesOut
	gmo.Messages = messages
	if len(messages) != 0 {
		smv := messages[len(messages)-1]
		if int(smv.Signet.LrcId) > 2 {
			cursor := strconv.Itoa(int(smv.Signet.LrcId))
			gmo.Cursor = &cursor
		}
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(gmo)
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
			did, err = atputils.TryLookupHandle(r.Context(), handle)
			if err != nil {
				h.serverError(w, errors.New("i think the handle might not exist?"+err.Error()))
				return
			}
			go h.db.StoreDidHandle(did, handle, context.Background())
		}
	}
	url := fmt.Sprintf("/lrc/%s/%s/ws", did, rkey)
	uri := fmt.Sprintf("at://%s/org.xcvr.feed.channel/%s", did, rkey)
	rchanres := types.ResolveChannelResponse{URL: url, URI: &uri}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(rchanres)
}

func (h *Handler) getProfileView(w http.ResponseWriter, r *http.Request) {
	handle := r.URL.Query().Get("handle")
	did := r.URL.Query().Get("did")
	if did == "" {
		if handle == "" {
			h.badRequest(w, errors.New("did not provide did or handle"))
			return
		}
		var err error
		did, err = h.db.ResolveHandle(handle, r.Context())
		if err != nil {
			did, err = atputils.TryLookupHandle(r.Context(), handle)
			if err != nil {
				h.serverError(w, errors.New("i think the handle might not exist?"+err.Error()))
				return
			}
			go h.db.StoreDidHandle(did, handle, context.Background())
		}
	}
	h.serveProfileView(did, handle, w, r)
}

func (h *Handler) serveProfileView(did string, handle string, w http.ResponseWriter, r *http.Request) {
	profile, err := h.db.GetProfileView(did, r.Context())
	if err != nil {
		h.notFound(w, errors.New(fmt.Sprintf("couldn't find profile for handle %s / did %s: %s", handle, did, err.Error())))
		return
	}
	profile.Handle = handle
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(profile)
}

func (h *Handler) getLastSeen(w http.ResponseWriter, r *http.Request) {
	handle := r.URL.Query().Get("handle")
	did := r.URL.Query().Get("did")
	if did == "" {
		if handle == "" {
			h.badRequest(w, errors.New("did not provide did or handle"))
			return
		}
		var err error
		did, err = h.db.ResolveHandle(handle, r.Context())
		if err != nil {
			did, err = atputils.TryLookupHandle(r.Context(), handle)
			if err != nil {
				h.serverError(w, errors.New("i think the handle might not exist?"+err.Error()))
				return
			}
			go h.db.StoreDidHandle(did, handle, context.Background())
		}
	}
	where, when := h.db.GetLastSeen(did, r.Context())
	type lastSeenResp struct {
		Where *string    `json:"where,omitempty"`
		When  *time.Time `json:"when,omitempty"`
	}
	resp := lastSeenResp{
		where, when,
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(resp)
}
