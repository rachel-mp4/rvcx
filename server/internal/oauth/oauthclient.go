package oauth

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/client"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/haileyok/atproto-oauth-golang"
	"github.com/haileyok/atproto-oauth-golang/helpers"

	"xcvr-backend/internal/db"
	"xcvr-backend/internal/lex"
	"xcvr-backend/internal/log"
	"xcvr-backend/internal/types"
)

type OauthXRPCClient struct {
	xrpc    *oauth.XrpcClient
	session *types.Session
	logger  *log.Logger
}

func NewOauthXRPCClient(s *db.Store, l *log.Logger, session *types.Session) *OauthXRPCClient {
	return &OauthXRPCClient{
		xrpc: &oauth.XrpcClient{
			OnDpopPdsNonceChanged: func(did, newNonce string) {
				err := s.SetDpopPdsNonce(session.ID, newNonce)
				if err != nil {
					l.Println(err.Error())
					return
				}
				session.DpopPdsNonce = newNonce
			},
		},
		session: session,
		logger:  l,
	}
}

func (c *OauthXRPCClient) getOauthSessionAuthArgs() (*oauth.XrpcAuthedRequestArgs, error) {
	s := c.session
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

func (c *OauthXRPCClient) MakeBskyPost(text string, ctx context.Context) error {
	authargs, err := c.getOauthSessionAuthArgs()
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
	err = c.xrpc.Do(ctx, authargs, "POST", "application/json", "com.atproto.repo.createRecord", nil, input, &out)
	if err != nil {
		return errors.New("oops! failed to make post: " + err.Error())
	}
	return nil
}

func (c *OauthXRPCClient) CreateXCVRProfile(profile lex.ProfileRecord, ctx context.Context) error {
	authargs, err := c.getOauthSessionAuthArgs()
	if err != nil {
		return errors.New("failed to get oauthsessionauthargs while making post: " + err.Error())
	}
	getOut, err := getProfileRecord(authargs.PdsUrl, authargs.Did, ctx)
	if err != nil {
		return errors.New("failed to getProfileRecord while creating XCVR profile: " + err.Error())
	}
	if getOut.Cid != nil {
		return errors.New("there already is a profileRecord, I don't want to overwrite it")
	}
	rkey := "self"
	input := atproto.RepoCreateRecord_Input{
		Collection: "org.xcvr.actor.profile",
		Repo:       authargs.Did,
		Rkey:       &rkey,
		Record:     &util.LexiconTypeDecoder{Val: &profile},
	}
	var out atproto.RepoCreateRecord_Output
	err = c.xrpc.Do(ctx, authargs, "POST", "application/json", "com.atproto.repo.createRecord", nil, input, &out)
	if err != nil {
		return errors.New("oops! failed to create a profile: " + err.Error())
	}
	return nil
}

func (c *OauthXRPCClient) CreateXCVRChannel(channel *lex.ChannelRecord, ctx context.Context) (uri string, cid string, err error) {
	authargs, err := c.getOauthSessionAuthArgs()
	if err != nil {
		err = errors.New("yikers! couldn't createXCVRChannel: " + err.Error())
		return
	}
	input := atproto.RepoCreateRecord_Input{
		Collection: "org.xcvr.feed.channel",
		Repo:       authargs.Did,
		Record:     &util.LexiconTypeDecoder{Val: channel},
	}
	var out atproto.RepoCreateRecord_Output
	err = c.xrpc.Do(ctx, authargs, "POST", "application/json", "com.atproto.repo.createRecord", nil, input, &out)
	if err != nil {
		err = errors.New("that's not good! failed to create a XCVRChannel: " + err.Error())
		return
	}
	uri = out.Uri
	cid = out.Cid
	return
}

func (c *OauthXRPCClient) CreateXCVRMessage(message lex.MessageRecord, ctx context.Context) (uri string, cid string, err error) {
	authargs, err := c.getOauthSessionAuthArgs()
	if err != nil {
		err = errors.New("uh oh... I couldn't make a XCVRMessage: " + err.Error())
		return
	}
	input := atproto.RepoCreateRecord_Input{
		Collection: "org.xcvr.lrc.message",
		Repo:       authargs.Did,
		Record:     &util.LexiconTypeDecoder{Val: &message},
	}
	var out atproto.RepoCreateRecord_Output
	err = c.xrpc.Do(ctx, authargs, "POST", "application/json", "com.atproto.repo.createRecord", nil, input, &out)
	if err != nil {
		err = errors.New("i've got a bad feeling aobut this... failed to create XCVRMessage: " + err.Error())
		return
	}
	uri = out.Uri
	cid = out.Cid
	return
}

func (c *OauthXRPCClient) UpdateXCVRProfile(profile lex.ProfileRecord, ctx context.Context) error {
	authargs, err := c.getOauthSessionAuthArgs()
	if err != nil {
		return errors.New("failed to get oauthsessionauthargs while making post: " + err.Error())
	}
	getOut, err := getProfileRecord(authargs.PdsUrl, authargs.Did, ctx)
	if err != nil {
		return errors.New("messed that up! " + err.Error())
	}
	if getOut.Cid == nil {
		return c.CreateXCVRProfile(profile, ctx)
	}
	rkey := "self"
	input := atproto.RepoPutRecord_Input{
		Collection: "org.xcvr.actor.profile",
		Repo:       authargs.Did,
		Rkey:       rkey,
		Record:     &util.LexiconTypeDecoder{Val: &profile},
		SwapRecord: getOut.Cid,
	}
	var out atproto.RepoPutRecord_Output
	err = c.xrpc.Do(ctx, authargs, "POST", "application/json", "com.atproto.repo.putRecord", nil, input, &out)
	if err != nil {
		return errors.New("oops! failed to update a profile: " + err.Error())
	}
	return nil
}

func getProfileRecord(pdsUrl string, did string, ctx context.Context) (*atproto.RepoGetRecord_Output, error) {
	cli := client.NewAPIClient(pdsUrl)
	return atproto.RepoGetRecord(ctx, cli, "", "org.xcvr.actor.profile", did, "self")
}
