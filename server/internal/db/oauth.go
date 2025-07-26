package db

import (
	"context"
	"errors"
	"fmt"
	"rvcx/internal/types"
)

func (s *Store) StoreOAuthRequest(req *types.OAuthRequest, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO oauthrequests (
		authserver_iss,
		state,
		did,
		pds_url,
		pkce_verifier,
		dpop_auth_server_nonce,
		dpop_private_jwk
		) VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		req.AuthserverIss,
		req.State,
		req.Did,
		req.PdsUrl,
		req.PkceVerifier,
		req.DpopAuthServerNonce,
		req.DpopPrivKey)
	return err
}

func (s *Store) StoreOAuthSession(session *types.Session, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO oauthsessions (
		authserver_iss,
		state,
		did,
		pds_url,
		pkce_verifier,
		dpop_auth_server_nonce,
		dpop_private_jwk,
		dpop_pds_nonce,
		access_token,
		refresh_token,
		expiration
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)`,
		session.AuthserverIss,
		session.State,
		session.Did,
		session.PdsUrl,
		session.PkceVerifier,
		session.DpopAuthServerNonce,
		session.DpopPrivKey,
		session.DpopPdsNonce,
		session.AccessToken,
		session.RefreshToken,
		session.Expiration)
	if err != nil {
		return errors.New("error storing oauth session" + err.Error())
	}
	return nil
}

func (s *Store) GetOauthRequest(state string, ctx context.Context) (*types.OAuthRequest, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT
			r.id,
			r.authserver_iss,
			r.did,
			r.pds_url,
			r.pkce_verifier,
			r.dpop_auth_server_nonce,
			r.dpop_private_jwk
		FROM oauthrequests r
		WHERE r.state = $1
		LIMIT 1
		`, state)
	var req types.OAuthRequest
	err := row.Scan(&req.ID, &req.AuthserverIss, &req.Did, &req.PdsUrl, &req.PkceVerifier, &req.DpopAuthServerNonce, &req.DpopPrivKey)
	if err != nil {
		return nil, errors.New("error scanning rows while getting oauth request:" + err.Error())
	}
	return &req, nil
}

func (s *Store) GetOauthSession(id int, ctx context.Context) (*types.Session, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT
			r.authserver_iss,
			r.did,
			r.pds_url,
			r.pkce_verifier,
			r.dpop_auth_server_nonce,
			r.dpop_private_jwk,
			r.dpop_pds_nonce,
			r.access_token,
			r.refresh_token,
			r.expiration
		FROM oauthsessions r
		WHERE r.id = $1
		`, id)
	var session types.Session
	err := row.Scan(
		&session.AuthserverIss,
		&session.Did,
		&session.PdsUrl,
		&session.PkceVerifier,
		&session.DpopAuthServerNonce,
		&session.DpopPrivKey,
		&session.DpopPdsNonce,
		&session.AccessToken,
		&session.RefreshToken,
		&session.Expiration)
	if err != nil {
		return nil, errors.New("error scanning oauthsession row: " + err.Error())
	}
	return &session, nil
}

func (s *Store) DeleteOauthRequest(state string, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM oauthrequests r WHERE r.state = $1
		`, state)
	if err != nil {
		return errors.New("error deleting oauth request:" + err.Error())
	}
	return nil
}

func (s *Store) DeleteOauthSession(id int, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM oauthsessions s WHERE s.id = $1
		`, id)
	if err != nil {
		return errors.New("error deleting oauth request:" + err.Error())
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
