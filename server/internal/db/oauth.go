package db

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (s Store) GetSession(ctx context.Context, did syntax.DID, sessionID string) (*oauth.ClientSessionData, error) {
	row := s.pool.QueryRow(ctx, `
	SELECT 
		host_url,
		authserver_url,
		authserver_token_endpoint,
		scopes,
		access_token,
		refresh_token,
		dpop_authserver_nonce,
		dpop_host_nonce,
		dpop_privatekey_multibase
	FROM sessions
	WHERE account_did = $1 AND session_id = $2`, did.String(), sessionID)
	var scope string
	var csd oauth.ClientSessionData
	csd.AccountDID = did
	csd.SessionID = sessionID
	err := row.Scan(&csd.HostURL,
		&csd.AuthServerURL,
		&csd.AuthServerTokenEndpoint,
		&scope,
		&csd.AccessToken,
		&csd.RefreshToken,
		&csd.DPoPAuthServerNonce,
		&csd.DPoPHostNonce,
		&csd.DPoPPrivateKeyMultibase,
	)
	if err != nil {
		return nil, errors.New("error scanning: " + err.Error())
	}
	scopes := strings.Fields(scope)
	csd.Scopes = scopes
	return &csd, nil
}

func (s Store) SaveSession(ctx context.Context, sess oauth.ClientSessionData) error {
	scope := strings.Join(sess.Scopes, " ")
	_, err := s.pool.Exec(ctx, `
		INSERT INTO sessions (
		session_id,
		account_did,
		host_url,
		authserver_url,
		authserver_token_endpoint,
		scopes,
		access_token,
		refresh_token,
		dpop_authserver_nonce,
		dpop_host_nonce,
		dpop_privatekey_multibase
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (session_id)
		DO UPDATE SET 
			account_did = EXCLUDED.account_did,
			host_url = EXCLUDED.host_url,
			authserver_url = EXCLUDED.authserver_url,
			authserver_token_endpoint = EXCLUDED.host_url,
			scopes = EXCLUDED.scopes,
			access_token = EXCLUDED.access_token,
			refresh_token = EXCLUDED.refresh_token,
			dpop_authserver_nonce = EXCLUDED.dpop_authserver_nonce,
			dpop_host_nonce = EXCLUDED.dpop_host_nonce,
			dpop_privatekey_multibase = EXCLUDED.privatekey_multibase
		`,
		sess.SessionID,
		sess.AccountDID.String(),
		sess.HostURL,
		sess.AuthServerURL,
		sess.AuthServerTokenEndpoint,
		scope,
		sess.AccessToken,
		sess.RefreshToken,
		sess.DPoPAuthServerNonce,
		sess.DPoPHostNonce,
		sess.DPoPPrivateKeyMultibase,
	)
	if err != nil {
		return errors.New("failed to insert: " + err.Error())
	}
	return nil
}

func (s Store) DeleteSession(ctx context.Context, did syntax.DID, sessionID string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM sessions WHERE account_did = $1 AND session_id = $2`, did.String(), sessionID)
	if err != nil {
		return errors.New("failed to delete: " + err.Error())
	}
	return nil
}

func (s Store) GetAuthRequestInfo(ctx context.Context, state string) (*oauth.AuthRequestData, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT 
  		authserver_url,
  		account_did,
  		scopes,
  		request_uri,
  		authserver_token_endpoint,
  		pkce_verifier,
  		dpop_authserver_nonce,
  		dpop_privatekey_multibase
  	FROM requests
		WHERE state = $1
		`, state)
	var ari oauth.AuthRequestData
	ari.State = state
	var did string
	err := row.Scan(
		&ari.AuthServerURL,
		&did,
		&ari.Scope,
		&ari.RequestURI,
		&ari.AuthServerTokenEndpoint,
		&ari.PKCEVerifier,
		&ari.DPoPAuthServerNonce,
		&ari.DPoPPrivateKeyMultibase,
	)
	if err != nil {
		return nil, errors.New("failed to scan: " + err.Error())
	}
	sdid, err := syntax.ParseDID(did)
	if err != nil {
		return nil, errors.New("failed to parse did: " + err.Error())
	}
	ari.AccountDID = &sdid
	return &ari, nil
}

func (s Store) SaveAuthRequestInfo(ctx context.Context, info oauth.AuthRequestData) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO requests (
			state,
  		authserver_url,
  		account_did,
  		scopes,
  		request_uri,
  		authserver_token_endpoint,
  		pkce_verifier,
  		dpop_authserver_nonce,
  		dpop_privatekey_multibase)
  	VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		info.State,
		info.AuthServerURL,
		info.AccountDID,
		info.Scope,
		info.RequestURI,
		info.AuthServerTokenEndpoint,
		info.PKCEVerifier,
		info.DPoPAuthServerNonce,
		info.DPoPPrivateKeyMultibase,
	)
	if err != nil {
		return errors.New("failed to insert: " + err.Error())
	}
	return nil
}

func (s Store) DeleteAuthRequestInfo(ctx context.Context, state string) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM requests WHERE state = $1`, state)
	if err != nil {
		return errors.New("failed to delete: " + err.Error())
	}
	return nil
}

func (s *Store) SetDpopPdsNonce(id int, dpopnonce string) error {
	_, err := s.pool.Exec(context.Background(), `
			UPDATE oauthsessions SET dpop_pds_nonce = $1 WHERE id = $2
		`, dpopnonce, id)
	if err != nil {
		return errors.New(fmt.Sprintf("error updating dpop nonce for id %d: %s", id, err.Error()))
	}
	return nil
}
