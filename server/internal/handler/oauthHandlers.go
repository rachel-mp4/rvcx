package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"xcvr-backend/internal/atputils"
	"xcvr-backend/internal/oauth"

	"github.com/gorilla/sessions"
	"github.com/haileyok/atproto-oauth-golang/helpers"
)

func (h *Handler) serveJWKS(w http.ResponseWriter, r *http.Request) {
	key, err := oauth.GetJWKS()
	if err != nil {
		h.serverError(w, err)
	}
	pubKey, err := (*key).PublicKey()
	if err != nil {
		h.serverError(w, err)
	}
	ro := helpers.CreateJwksResponseObject(pubKey)
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(ro)
}

func (h *Handler) oauthLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		h.badRequest(w, err)
		return
	}
	handle := r.FormValue("handle")
	req, res, err := h.oauth.StartAuthFlow(r.Context(), handle)
	if err != nil {
		h.serverError(w, err)
		return
	}
	err = h.db.StoreOAuthRequest(req, r.Context())
	if err != nil {
		h.serverError(w, err)
		return
	}
	u, _ := url.Parse(res.AuthzEndpoint)
	u.RawQuery = fmt.Sprintf("client_id=%s&request_uri=%s", url.QueryEscape(oauth.GetClientMetadata().ClientId), res.RequestUri)

	session, _ := h.sessionStore.Get(r, "oauthsession")
	session.Values = map[any]any{}

	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   300,
		HttpOnly: true,
	}
	session.Values["oauth_state"] = res.State
	session.Values["oauth_did"] = res.DID
	err = session.Save(r, w)
	if err != nil {
		h.serverError(w, err)
		return
	}
	go func() {
		err := h.db.StoreDidHandle(res.DID, handle, context.Background())
		h.logger.Deprintln("storing....")
		if err != nil {
			h.logger.Deprintln("failed to store did handle: " + err.Error())
		}
		err = h.db.InitializeProfile(res.DID, handle, context.Background())
		h.logger.Deprintln("initializing....")
		if err != nil {
			h.logger.Deprintln("failed to initialize profile: " + err.Error())
		}
	}()
	http.Redirect(w, r, u.String(), http.StatusFound)
}

func (h *Handler) oauthCallback(w http.ResponseWriter, r *http.Request) {
	resState := r.FormValue("state")
	resIss := r.FormValue("iss")
	resCode := r.FormValue("code")
	session, err := h.sessionStore.Get(r, "oauthsession")
	if err != nil {
		h.serverError(w, err)
		return
	}
	if resState == "" || resIss == "" || resCode == "" {
		h.badRequest(w, errors.New("did not provide one of resState, resIss, resCode"))
		return
	}
	sessionState, ok := session.Values["oauth_state"].(string)
	if !ok {
		h.serverError(w, errors.New("oauth_state not found in session"))
		return
	}
	if resState != sessionState {
		h.serverError(w, errors.New("resState and sessionState do not match!"))
		return
	}
	params := oauth.CallbackParams{
		State: resState,
		Iss:   resIss,
		Code:  resCode,
	}
	req, err := h.db.GetOauthRequest(resState, r.Context())
	if err != nil {
		h.serverError(w, err)
		return
	}
	OauthSession, err := h.oauth.OauthCallback(r.Context(), req, params)
	if err != nil {
		h.serverError(w, err)
		return
	}
	err = h.db.DeleteOauthRequest(resState, r.Context())
	if err != nil {
		h.serverError(w, err)
		return
	}
	err = h.db.StoreOAuthSession(OauthSession, r.Context())
	if err != nil {
		h.serverError(w, err)
		return
	}

	session.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   86400 * 7,
		HttpOnly: true,
	}
	session.Values = map[any]any{}
	session.Values["did"] = req.Did
	err = session.Save(r, w)
	if err != nil {
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, "/", http.StatusFound)
}

func oauthCallbackPath() string {
	mp := os.Getenv("MY_OAUTH_CALLBACK")
	return fmt.Sprintf("GET %s", mp)
}

func oauthJWKSPath() string {
	mp := os.Getenv("MY_JWKS_PATH")
	return fmt.Sprintf("GET %s", mp)
}

func (h *Handler) getSession(w http.ResponseWriter, r *http.Request) {
	did, handle, err := h.findDidAndHandle(w, r)
	if err != nil {
		h.handleFindDidAndHandleError(w, r, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"did":    did,
		"handle": handle,
	})
}

func (h *Handler) findDidAndHandle(w http.ResponseWriter, r *http.Request) (string, string, error) {
	session, _ := h.sessionStore.Get(r, "oauthsession")
	did, ok := session.Values["did"].(string)
	if !ok || did == "" {
		return "", "", errors.New("not authenticated")
	}
	handle, err := h.db.ResolveDid(did, r.Context())
	if err != nil {
		handle, err = atputils.GetHandleFromDid(r.Context(), did)
		if err != nil {
			return "", "", errors.New("error resolving handle" + err.Error())
		}
		h.logger.Deprintln("storing...")
		err = h.db.StoreDidHandle(did, handle, r.Context())
		if err != nil {
			h.logger.Deprintln("error storing did_handle in findDidAndHandle: " + err.Error())
		}
	}
	return did, handle, nil
}

func (h *Handler) handleFindDidAndHandleError(w http.ResponseWriter, r *http.Request, err error) {
	if err != nil {
		if err.Error() == "not authenticated" {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}
		h.serverError(w, err)
		return
	}
	h.logger.Deprintln("handling nil error?")
}
