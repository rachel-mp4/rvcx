package oauth

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/haileyok/atproto-oauth-golang"
	"github.com/haileyok/atproto-oauth-golang/helpers"

	"xcvr-backend/internal/db"
	"xcvr-backend/internal/lex"
	"xcvr-backend/internal/log"
	"xcvr-backend/internal/types"
)

type Client struct {
	xrpccli *oauth.XrpcClient
}

func NewXRPCClient(s *db.Store, l *log.Logger) *Client {
	return &Client{
		xrpccli: &oauth.XrpcClient{
			OnDpopPdsNonceChanged: func(did, newNonce string) {
				err := s.SetDpopPdsNonce(did, newNonce)
				if err != nil {
					l.Deprintln(err.Error())
				}
			},
		},
	}
}

func getOauthSessionAuthArgs(s *types.Session) (*oauth.XrpcAuthedRequestArgs, error) {
	privateJwk, err := helpers.ParseJWKFromBytes([]byte(s.DpopPrivKey))
	if err != nil {
		return nil, errors.New("failed to parse jwk in getoauthsessionauthargs: " + err.Error())
	}
	return &oauth.XrpcAuthedRequestArgs{
		Did:            s.Did,
		AccessToken:    s.AccessToken,
		PdsUrl:         s.PdsUrl,
		Issuer:         s.AuthserverIss,
		DpopPdsNonce:   s.DpopPdsNonce,
		DpopPrivateJwk: privateJwk,
	}, nil
}

func (c *Client) MakeBskyPost(text string, s *types.Session, ctx context.Context) error {
	authargs, err := getOauthSessionAuthArgs(s)
	if err != nil {
		return errors.New("failed to get oauthsessionauthargs while making post: " + err.Error())
	}
	post := bsky.FeedPost{
		Text:      text,
		CreatedAt: syntax.DatetimeNow().String(),
	}
	input := atproto.RepoCreateRecord_Input{
		Collection: "app.bsky.feed.post",
		Repo:       authargs.Did,
		Record:     &util.LexiconTypeDecoder{Val: &post},
	}
	var out atproto.RepoCreateRecord_Output
	err = c.xrpccli.Do(ctx, authargs, "POST", "application/json", "com.atproto.repo.createRecord", nil, input, &out)
	if err != nil {
		return errors.New("oops! failed to make post: " + err.Error())
	}
	return nil
}

func (c *Client) CreateXCVRProfile(profile lex.ProfileRecord, s *types.Session, ctx context.Context) error {
	authargs, err := getOauthSessionAuthArgs(s)
	if err != nil {
		return errors.New("failed to get oauthsessionauthargs while making post: " + err.Error())
	}
	rkey := "self"
	input := atproto.RepoCreateRecord_Input{
		Collection: "org.xcvr.actor.profile",
		Repo:       authargs.Did,
		Rkey:       &rkey,
		Record:     &util.LexiconTypeDecoder{Val: &profile},
	}
	var out atproto.RepoCreateRecord_Output
	err = c.xrpccli.Do(ctx, authargs, "POST", "application/json", "com.atproto.repo.createRecord", nil, input, &out)
	if err != nil {
		return errors.New("oops! failed to create a profile: " + err.Error())
	}
	return nil
}

func (c *Client) UpdateXCVRProfile(profile lex.ProfileRecord, s *types.Session, ctx context.Context) error {
	authargs, err := getOauthSessionAuthArgs(s)
	if err != nil {
		return errors.New("failed to get oauthsessionauthargs while making post: " + err.Error())
	}
	rkey := "self"
	input := atproto.RepoPutRecord_Input{
		Collection: "org.xcvr.actor.profile",
		Repo:       authargs.Did,
		Rkey:       rkey,
		Record:     &util.LexiconTypeDecoder{Val: &profile},
	}
	var out atproto.RepoPutRecord_Output
	err = c.xrpccli.Do(ctx, authargs, "POST", "application/json", "com.atproto.repo.putRecord", nil, input, &out)
	if err != nil {
		return errors.New("oops! failed to update a profile: " + err.Error())
	}
	return nil
}
