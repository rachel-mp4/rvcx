package oauth

import (
	"context"
	"errors"
	"fmt"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/api/bsky"
	"github.com/bluesky-social/indigo/atproto/client"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/indigo/lex/util"
	"github.com/haileyok/atproto-oauth-golang"
	"github.com/haileyok/atproto-oauth-golang/helpers"
	"os"

	"xcvr-backend/internal/db"
	"xcvr-backend/internal/lex"
	"xcvr-backend/internal/log"
	"xcvr-backend/internal/types"
)

type Client struct {
	xrpccli    *oauth.XrpcClient
	xcvrcli    *client.APIClient
	accessjwt  *string
	refreshjwt *string
	did        *string
	logger     *log.Logger
}

func NewXRPCClient(s *db.Store, l *log.Logger, host string, did string) *Client {
	return &Client{
		xrpccli: &oauth.XrpcClient{
			OnDpopPdsNonceChanged: func(did, newNonce string) {
				err := s.SetDpopPdsNonce(did, newNonce)
				if err != nil {
					l.Deprintln(err.Error())
				}
			},
		}, xcvrcli: client.NewAPIClient(host), did: &did,
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
	err = c.xrpccli.Do(ctx, authargs, "POST", "application/json", "com.atproto.repo.createRecord", nil, input, &out)
	if err != nil {
		return errors.New("oops! failed to create a profile: " + err.Error())
	}
	return nil
}

func (c *Client) CreateXCVRChannel(channel lex.ChannelRecord, s *types.Session, ctx context.Context) error {
	authargs, err := getOauthSessionAuthArgs(s)
	if err != nil {
		return errors.New("yikers! couldn't createXCVRChannel: " + err.Error())
	}
	input := atproto.RepoCreateRecord_Input{
		Collection: "org.xcvr.feed.channel",
		Repo:       authargs.Did,
		Record:     &util.LexiconTypeDecoder{Val: &channel},
	}
	var out atproto.RepoCreateRecord_Output
	err = c.xrpccli.Do(ctx, authargs, "POST", "application/json", "com.atproto.repo.createRecord", nil, input, &out)
	if err != nil {
		return errors.New("that's not good! failed to create a XCVRChannel: " + err.Error())
	}
	return nil
}

func (c *Client) CreateXCVRMessage(message lex.MessageRecord, s *types.Session, ctx context.Context) error {
	authargs, err := getOauthSessionAuthArgs(s)
	if err != nil {
		return errors.New("uh oh... I couldn't make a XCVRMessage: " + err.Error())
	}
	input := atproto.RepoCreateRecord_Input{
		Collection: "org.xcvr.lrc.message",
		Repo:       authargs.Did,
		Record:     &util.LexiconTypeDecoder{Val: &message},
	}
	var out atproto.RepoCreateRecord_Output
	err = c.xrpccli.Do(ctx, authargs, "POST", "application/json", "com.atproto.repo.createRecord", nil, input, &out)
	if err != nil {
		return errors.New("i've got a bad feeling aobut this... failed to create XCVRMessage: " + err.Error())
	}
	return nil
}

func (c *Client) CreateSession(ctx context.Context) error {
	c.logger.Deprintln("creating session...")
	secret := os.Getenv("MY_SECRET")
	identity := os.Getenv("MY_IDENTITY")
	params := map[string]any{
		"identifier": &identity,
		"password":   &secret,
	}
	var out atproto.ServerCreateSession_Output
	err := c.xcvrcli.LexDo(ctx, "POST", "application/json", "com.atproto.server.createSession", params, nil, out)
	if err != nil {
		return errors.New("I couldn't create a session: " + err.Error())
	}
	c.accessjwt = &out.AccessJwt
	c.refreshjwt = &out.RefreshJwt
	c.logger.Deprintln("created session!")
	return nil
}

func (c *Client) RefreshSession(ctx context.Context) error {
	c.logger.Deprintln("refreshing session")
	c.xcvrcli.Headers.Set("Authorization", fmt.Sprintf("Bearer %s", *c.refreshjwt))
	var out atproto.ServerRefreshSession_Output
	err := c.xcvrcli.LexDo(ctx, "POST", "application/json", "com.atproto.server.refreshSession", nil, nil, out)
	if err != nil {
		c.logger.Println("FAILED TO REFRESH RESSION")
		return errors.New("failed to refresh session! " + err.Error())
	}
	c.accessjwt = &out.AccessJwt
	c.refreshjwt = &out.RefreshJwt
	c.logger.Deprintln("refreshed session!")
	return nil
}

func (c *Client) CreateXCVRSignet(signet lex.SignetRecord, ctx context.Context) error {
	if c.accessjwt == nil {
		return errors.New("must create a session first")
	}
	c.xcvrcli.Headers.Set("Authorization", fmt.Sprintf("Bearer %s", *c.accessjwt))
	input := atproto.RepoCreateRecord_Input{
		Collection: "org.xcvr.lrc.signet",
		Repo:       *c.did,
		Record:     &util.LexiconTypeDecoder{Val: &signet},
	}
	var out atproto.RepoCreateRecord_Output
	err := c.xcvrcli.LexDo(ctx, "POST", "application/json", "com.atproto.repo.createRecord", nil, input, out)
	if err != nil {
		err1 := err.Error()
		err = c.RefreshSession(ctx)
		if err != nil {
			return errors.New("failed to refresh session while creating signet! first " + err1 + " then " + err.Error())
		}
		c.xcvrcli.Headers.Set("Authorization", fmt.Sprintf("Bearer %s", *c.accessjwt))
		out = atproto.RepoCreateRecord_Output{}
		err = c.xcvrcli.LexDo(ctx, "POST", "application/json", "com.atproto.repo.createRecord", nil, input, out)
		if err != nil {
			return errors.New("not good, failed to create signet after failing then refreshing session! first " + err1 + " then " + err.Error())
		}
	}
	return nil
}

func (c *Client) UpdateXCVRProfile(profile lex.ProfileRecord, s *types.Session, ctx context.Context) error {
	authargs, err := getOauthSessionAuthArgs(s)
	if err != nil {
		return errors.New("failed to get oauthsessionauthargs while making post: " + err.Error())
	}
	getOut, err := getProfileRecord(authargs.PdsUrl, authargs.Did, ctx)
	if err != nil {
		return errors.New("messed that up! " + err.Error())
	}
	if getOut.Cid == nil {
		return c.CreateXCVRProfile(profile, s, ctx)
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
	err = c.xrpccli.Do(ctx, authargs, "POST", "application/json", "com.atproto.repo.putRecord", nil, input, &out)
	if err != nil {
		return errors.New("oops! failed to update a profile: " + err.Error())
	}
	return nil
}

func getProfileRecord(pdsUrl string, did string, ctx context.Context) (*atproto.RepoGetRecord_Output, error) {
	cli := client.NewAPIClient(pdsUrl)
	return atproto.RepoGetRecord(ctx, cli, "", "org.xcvr.actor.profile", did, "self")
}
