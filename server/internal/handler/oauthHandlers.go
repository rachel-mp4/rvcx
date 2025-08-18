package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"rvcx/internal/atputils"
	"rvcx/internal/oauth"
	"strings"

	"github.com/gorilla/sessions"
)

func (h *Handler) serveJWKS(w http.ResponseWriter, r *http.Request) {
	key, err := oauth.GetPrivateKey()
	if err != nil {
		h.serverError(w, err)
	}
	pubKey, err := (*key).PublicKey()
	if err != nil {
		h.serverError(w, err)
	}
	ro, err := pubKey.JWK()
	if err != nil {
		h.serverError(w, err)
	}
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
	identifier := r.FormValue("identifier")
	redirectURL, err := h.oauth.StartAuthFlow(r.Context(), identifier)
	if err != nil {
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *Handler) oauthCallback(w http.ResponseWriter, r *http.Request) {
	sessData, err := h.oauth.OauthCallback(r.Context(), r.URL.Query())
	err = h.rm.CreateInitialProfile(sessData, r.Context())
	if err != nil {
		h.serverError(w, err)
		return
	}
	session, _ := h.sessionStore.Get(r, "oauthsession")
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
	session.Values["did"] = sessData.AccountDID.String()
	session.Values["id"] = sessData.SessionID
	session.Values["scopes"] = strings.Join(sessData.Scopes, " ")
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
	did, handle, err := h.findDidAndHandle(r)
	if err != nil {
		h.handleFindDidAndHandleError(w, err)
		return
	}
	h.serveProfileView(did, handle, w, r)
}

func (h *Handler) findDidAndHandle(r *http.Request) (string, string, error) {
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

func (h *Handler) handleFindDidAndHandleError(w http.ResponseWriter, err error) {
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

func (h *Handler) oauthLogout(w http.ResponseWriter, r *http.Request) {
	s, _ := h.sessionStore.Get(r, "oauthsession")
	id, ok := s.Values["id"].(string)
	did, bok := s.Values["did"].(string)
	if ok && bok {
		h.logger.Deprintln("deleting session to log out!")
		err := h.rm.DeleteSession(did, id, r.Context())
		if err != nil {
			h.serverError(w, errors.New("couldn't log out: "+err.Error()))
			return
		}
		h.logger.Deprintln("deleted session to log out!")
	}
	s.Values = make(map[any]any)
	s.Options.MaxAge = -1
	h.logger.Deprintln("saving cookie to log out!")
	err := s.Save(r, w)
	if err != nil {
		h.serverError(w, errors.New("issue logging out: "+err.Error()))
		return
	}
	h.logger.Deprintln("saved cookie to log out!")
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
