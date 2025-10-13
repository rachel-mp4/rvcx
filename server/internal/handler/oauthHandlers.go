package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"rvcx/internal/atputils"
	"rvcx/internal/oauth"
	"strconv"
	"strings"
	"time"

	atoauth "github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/gorilla/sessions"
)

func (h *Handler) serveJWKS(w http.ResponseWriter, r *http.Request) {
	key, err := oauth.GetPrivateKey()
	if err != nil {
		h.serverError(w, err)
	}
	pubKey, err := key.PublicKey()
	if err != nil {
		h.serverError(w, err)
	}
	ro, err := pubKey.JWK()
	if err != nil {
		h.serverError(w, err)
	}

	cski := os.Getenv("CLIENT_SECRET_KEY_ID")
	ro.KeyID = &cski
	rro := map[string]any{"keys": []any{ro}}
	w.Header().Set("Content-Type", "application/json")
	encoder := json.NewEncoder(w)
	encoder.Encode(rro)
}

func (h *Handler) oauthLogin(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		h.badRequest(w, err)
		return
	}
	identifier := strings.TrimSpace(r.FormValue("identifier"))
	redirectURL, err := h.oauth.StartAuthFlow(r.Context(), identifier)
	if err != nil {
		h.serverError(w, err)
		return
	}
	http.Redirect(w, r, redirectURL, http.StatusFound)
}

func (h *Handler) oauthCallback(w http.ResponseWriter, r *http.Request) {
	sessData, err := h.oauth.OauthCallback(r.Context(), r.URL.Query())
	if err != nil {
		h.serverError(w, errors.New("my god.... :"+err.Error()))
		return
	}
	isban, err := h.db.IsBanned(sessData.AccountDID.String(), r.Context())
	if err != nil {
		h.serverError(w, errors.New("i'm not sure if user is banned, error, "+err.Error()))
		return
	}
	if isban {
		ban, _ := h.db.GetBanned(sessData.AccountDID.String(), r.Context())
		http.Redirect(w, r, fmt.Sprintf("%s%d", os.Getenv("BAN_ENDPOINT"), ban.Id), http.StatusSeeOther)
		return
	}

	err = h.rm.CreateInitialProfile(sessData, r.Context())
	if err != nil {
		h.serverError(w, err)
		return
	}
	session, _ := h.sessionStore.Get(r, "oauthsession")

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
	s, _ := h.sessionStore.Get(r, "oauthsession")
	did, ok := s.Values["did"].(string)
	if !ok {
		h.notFound(w, errors.New("couldn't find profile"))
		return
	}
	handle, err := h.db.FullResolveDid(did, r.Context())
	if err != nil {
		h.notFound(w, errors.New("coudln't resolve did: "+err.Error()))
		return
	}
	h.serveProfileView(did, handle, w, r)
}

func (h *Handler) oauthLogout(cs *atoauth.ClientSession, w http.ResponseWriter, r *http.Request) {
	if cs != nil {
		h.logger.Deprintln("deleting session to log out!")
		err := h.db.DeleteSession(r.Context(), cs.Data.AccountDID, cs.Data.SessionID)
		if err != nil {
			h.serverError(w, errors.New("couldn't log out: "+err.Error()))
			return
		}
		h.logger.Deprintln("deleted session to log out!")
	}

	s, _ := h.sessionStore.Get(r, "oauthsession")
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

func (h *Handler) oauthMiddleware(f func(cs *atoauth.ClientSession, w http.ResponseWriter, r *http.Request)) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		s, _ := h.sessionStore.Get(r, "oauthsession")
		id, ok := s.Values["id"].(string)
		did, bok := s.Values["did"].(string)
		if !ok || !bok {
			f(nil, w, r)
			return
		}
		sdid, err := syntax.ParseDID(did)
		if err != nil {
			f(nil, w, r)
			return
		}
		cs, err := h.oauth.ResumeSession(r.Context(), sdid, id)
		if err != nil {
			f(nil, w, r)
			return
		}
		f(cs, w, r)
	}
}

func (h *Handler) postBan(w http.ResponseWriter, r *http.Request) {
	s, _ := h.sessionStore.Get(r, "oauthsession")
	did, bok := s.Values["did"].(string)
	if !bok {
		h.badRequest(w, errors.New("not authorized"))
		return
	}
	handle, err := h.db.ResolveDid(did, r.Context())
	if err != nil {
		h.serverError(w, errors.New("failed to resolve"+err.Error()))
		return
	}
	if handle != os.Getenv("ADMIN_HANDLE") {
		h.badRequest(w, errors.New("must be admin to ban"))
		return
	}
	err = r.ParseForm()
	if err != nil {
		h.badRequest(w, err)
	}
	userhandle := r.FormValue("user")
	userdid, err := atputils.GetDidFromHandle(r.Context(), userhandle)
	if err != nil {
		h.badRequest(w, errors.New("failed to resolve user handle"))
		return
	}
	daysstring := r.FormValue("days")
	daysint, err := strconv.Atoi(daysstring)
	var till *time.Time
	if err == nil {
		tillt := time.Now().Add(time.Hour * 24 * time.Duration(daysint))
		till = &tillt
	}
	var reason *string
	reasonstr := r.FormValue("reason")
	if reasonstr != "" {
		reason = &reasonstr
	}
	err = h.db.AddBan(userdid, reason, till, r.Context())
	if err != nil {
		h.serverError(w, errors.New("failed to ban, "+err.Error()))
		return
	}
	ban, err := h.db.GetBanned(userdid, r.Context())
	if err != nil {
		h.serverError(w, errors.New("succeeded to ban and then failed again"+err.Error()))
		return
	}
	err = h.db.DeleteAllSessions(r.Context(), ban.Did)
	if err != nil {
		h.serverError(w, errors.New("failed to kick user "+ban.Did+err.Error()))
		return
	}
	w.Header().Set("Location", fmt.Sprintf("%s%d", os.Getenv("BAN_ENDPOINT"), ban.Id))
	w.WriteHeader(http.StatusCreated)
	w.Write(nil)
}

func (h *Handler) getBan(w http.ResponseWriter, r *http.Request) {
	banid := r.URL.Query().Get("id")
	id, err := strconv.Atoi(banid)
	if err != nil {
		h.badRequest(w, err)
		return
	}
	ban, err := h.db.GetBanId(id, r.Context())
	if err != nil {
		h.serverError(w, err)
		return
	}
	encoder := json.NewEncoder(w)
	w.Header().Add("Content-Type", "application/json")
	encoder.Encode(ban)
}
