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
	"strings"

	atoauth "github.com/bluesky-social/indigo/atproto/auth/oauth"
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

func (h *Handler) postChannel(cs *atoauth.ClientSession, w http.ResponseWriter, r *http.Request) {
	cr, err := h.parseChannelRequest(r)
	if err != nil {
		h.badRequest(w, err)
		return
	}
	var did, uri string
	if cs == nil {
		did, uri, err = h.rm.PostMyChannel(r.Context(), cr)
	} else {
		did, uri, err = h.rm.PostChannel(cs, r.Context(), cr)
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

func (h *Handler) postMessage(cs *atoauth.ClientSession, w http.ResponseWriter, r *http.Request) {
	pmr, err := h.parseMessageRequest(r)
	if err != nil {
		h.badRequest(w, errors.New("failed to parse message request: "+err.Error()))
		return
	}
	if cs == nil {
		err = h.rm.PostMyMessage(r.Context(), pmr)
	} else {
		err = h.rm.PostMessage(cs, r.Context(), pmr)
	}
	if err != nil {
		h.serverError(w, errors.New("error posting message: "+err.Error()))
		return
	}
	w.Write(nil)
}

func (h *Handler) postMyMessage(w http.ResponseWriter, r *http.Request) {
	pmr, err := h.parseMessageRequest(r)
	if err != nil {
		h.badRequest(w, errors.New("failed to parse message request: "+err.Error()))
		return
	}
	err = h.rm.PostMyMessage(r.Context(), pmr)
	if err != nil {
		h.serverError(w, errors.New("error posting message: "+err.Error()))
	}
	w.Write(nil)
}

func (h *Handler) deleteChannel(cs *atoauth.ClientSession, w http.ResponseWriter, r *http.Request) {
	if cs == nil {
		h.logger.Deprintln("tried to anonymously delete")
		return
	}
	rkey := r.PathValue("rkey")
	user := r.PathValue("user")
	var err error
	if cs.Data.AccountDID.String() == user {
		err = h.rm.DeleteChannel(cs, rkey, r.Context())
	} else if cs.Data.AccountDID.String() == os.Getenv("ADMIN_DID") {
		uri := fmt.Sprintf("at://%s/org.xcvr.feed.channel/%s", user, rkey)
		err = h.rm.AcceptChannelDelete(uri, r.Context())
	}
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

func (h *Handler) uploadImage(cs *atoauth.ClientSession, w http.ResponseWriter, r *http.Request) {
	if cs == nil {
		h.badRequest(w, errors.New("must be authorized to post image"))
		return
	}
	err := r.ParseMultipartForm(1 << 20)
	if err != nil {
		h.badRequest(w, errors.New("beep bop bad image: "+err.Error()))
		return
	}
	file, fheader, err := r.FormFile("image")
	if err != nil {
		h.badRequest(w, errors.New("failed to formfile: "+err.Error()))
		return
	}
	defer file.Close()
	ct := fheader.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "image/") {
		h.badRequest(w, errors.New("must post an image"))
		return
	}
	blob, err := h.rm.PostImage(cs, file, r.Context())
	if err != nil {
		h.serverError(w, errors.New("failed to upload: "+err.Error()))
		return
	}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(blob)
}

func (h *Handler) postMedia(cs *atoauth.ClientSession, w http.ResponseWriter, r *http.Request) {
	if cs == nil {
		h.badRequest(w, errors.New("must be authorized to post media"))
	}
	mr, err := parseMediaRequest(r)
	if err != nil {
		h.badRequest(w, err)
		return
	}
	err = h.rm.PostMedia(cs, mr, r.Context())
	if err != nil {
		h.serverError(w, errors.New("failing to post the media :c"))
		return
	}
	w.Write(nil)
}

func parseMediaRequest(r *http.Request) (*types.ParseMediaRequest, error) {
	beep := json.NewDecoder(r.Body)
	var mr types.ParseMediaRequest
	err := beep.Decode(&mr)
	if err != nil {
		return nil, errors.New("A aaaaaa : " + err.Error())
	}
	return &mr, nil
}

// func (h *Handler) getImage(w http.ResponseWriter, r *http.Request) {
// 	vals := r.URL.Query()
// 	uri := vals.Get("uri")
// 	if uri == "" {
// 		h.badRequest(w, errors.New("must provide a did and cid"))
// 		return
// 	}
// 	image, err := h.db.GetImage(uri, r.Context())
// 	if err != nil {
// 		h.notFound(w, err)
// 		return
// 	}
// 	uploadDir := fmt.Sprintf("./uploads/%s", image.DID)
// 	_, err = os.Stat(uploadDir)
// 	if os.IsNotExist(err) {
// 		os.Mkdir(uploadDir, 0755)
// 	}
//
// 	imgPath := fmt.Sprintf("./uploads/%s", image.ImageCID)
// 	_, err = os.Stat(imgPath)
// 	if err != nil {
// 		syncGetBlob(image.DID, image.ImageCID)
// 	}
//
// 	img, err := os.Open(fmt.Sprintf("%s/%s", uploadDir, image.ImageCID))
// 	img.WriteTo(w)
// }

// func syncGetBlob(did string, cid *string) {
// 	//TODO: impl
// }
