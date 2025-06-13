package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gorilla/sessions"
	"net/http"
	"net/url"
	"os"
	"xcvr-backend/internal/oauth"
	"github.com/haileyok/atproto-oauth-golang/helpers"
)

func (h *Handler) serveJWKS(w http.ResponseWriter, r *http.Request) {
	pubKey, err := oauth.GetJWKS()
	if err != nil {
		h.serverError(w, err)
	}
	ro := helpers.CreateJwksResponseObject(*pubKey)
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
	session, _ := h.sessionStore.Get(r, "oauthsession")
	did, ok := session.Values["did"].(string)
	if !ok || did == "" {
		http.Error(w, "not authenticated", http.StatusUnauthorized)
		return
	}
	json.NewEncoder(w).Encode(map[string]any{
		"did": did,
	})
}
