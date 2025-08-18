package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"

	"rvcx/internal/lex"
	"rvcx/internal/log"
	"rvcx/internal/types"
)

type OauthXRPCClient struct {
	session *types.Session
	logger  *log.Logger
}

func (c *OauthXRPCClient) GetSession() *types.Session {
	return c.session
}

func MakeBskyPost(cs *oauth.ClientSession, text string, ctx context.Context) error {
	c := cs.APIClient()
	body := map[string]any{
		"repo":       *c.AccountDID,
		"collection": "app.bsky.feed.post",
		"record": map[string]any{
			"$type":     "app.bsky.feed.post",
			"text":      text,
			"createdAt": syntax.DatetimeNow(),
		},
	}
	err := c.Post(ctx, "com.atproto.repo.createRecord", body, nil)
	if err != nil {
		return errors.New("failed to tweet: " + err.Error())
	}
	return nil
}

func CreateXCVRProfile(cs *oauth.ClientSession, profile *lex.ProfileRecord, ctx context.Context) (p *lex.ProfileRecord, err error) {
	c := cs.APIClient()
	nsid, err := syntax.ParseNSID("com.atproto.repo.getRecord")
	if err != nil {
		return nil, errors.New("failed to parse: " + err.Error())
	}
	var getOut atproto.RepoGetRecord_Output
	body := map[string]any{
		"collection": "org.xcvr.actor.profile",
		"repo":       *c.AccountDID,
		"rkey":       "self",
	}
	err = c.Get(ctx, nsid, body, &getOut)
	if err == nil {
		if getOut.Cid != nil {
			var jsonBytes []byte
			jsonBytes, err = json.Marshal(getOut.Value)
			if err != nil {
				return
			}
			var pro lex.ProfileRecord
			err = json.Unmarshal(jsonBytes, &pro)
			if err != nil {
				return
			}
			return &pro, nil
		}
	}
	body["record"] = profile
	var out atproto.RepoCreateRecord_Output
	err = c.Post(ctx, "com.atproto.repo.createRecord", body, out)
	if err != nil {
		err = errors.New("oops! failed to create a profile: " + err.Error())
		return
	}
	return profile, nil
}

func CreateXCVRChannel(cs *oauth.ClientSession, channel *lex.ChannelRecord, ctx context.Context) (uri string, cid string, err error) {
	c := cs.APIClient()
	body := map[string]any{
		"collection": "org.xcvr.feed.channel",
		"repo":       *c.AccountDID,
		"record":     channel,
	}
	var out atproto.RepoCreateRecord_Output
	err = c.Post(ctx, "com.atproto.repo.createRecord", body, out)
	if err != nil {
		err = errors.New("oops! failed to create a channel: " + err.Error())
		return
	}
	uri = out.Uri
	cid = out.Cid
	return
}

func DeleteXCVRChannel(cs *oauth.ClientSession, rkey string, ctx context.Context) error {
	c := cs.APIClient()
	var getOut atproto.RepoGetRecord_Output
	body := map[string]any{
		"collection": "org.xcvr.feed.channel",
		"repo":       *c.AccountDID,
		"rkey":       rkey,
	}
	err := c.Get(ctx, "com.atproto.repo.getRecord", body, &getOut)
	if err != nil {
		return err
	}
	if getOut.Cid == nil {
		return nil
	}
	body["swapRecord"] = getOut.Cid
	err = c.Post(ctx, "com.atproto.repo.deleteRecord", body, nil)
	if err != nil {
		return err
	}
	return nil
}

func CreateXCVRMessage(cs *oauth.ClientSession, message *lex.MessageRecord, ctx context.Context) (uri string, cid string, err error) {
	c := cs.APIClient()
	body := map[string]any{
		"collection": "org.xcvr.lrc.message",
		"repo":       *c.AccountDID,
		"record":     message,
	}
	var out atproto.RepoCreateRecord_Output
	err = c.Post(ctx, "com.atproto.repo.createRecord", body, out)
	if err != nil {
		err = errors.New("oops! failed to create a message: " + err.Error())
		return
	}
	uri = out.Uri
	cid = out.Cid
	return
}

func UpdateXCVRProfile(cs *oauth.ClientSession, profile *lex.ProfileRecord, ctx context.Context) (p *lex.ProfileRecord, err error) {
	c := cs.APIClient()
	nsid, err := syntax.ParseNSID("com.atproto.repo.getRecord")
	if err != nil {
		return nil, errors.New("failed to parse: " + err.Error())
	}
	var getOut atproto.RepoGetRecord_Output
	err = c.Get(ctx, nsid, nil, &getOut)
	if err == nil {
		if getOut.Cid != nil {
			var jsonBytes []byte
			jsonBytes, err = json.Marshal(getOut.Value)
			if err != nil {
				return
			}
			var pro lex.ProfileRecord
			err = json.Unmarshal(jsonBytes, &pro)
			if err != nil {
				return
			}
			return &pro, nil
		}
	}
	body := map[string]any{
		"collection": "org.xcvr.actor.profile",
		"repo":       *c.AccountDID,
		"rkey":       "self",
		"record":     profile,
	}
	var out atproto.RepoCreateRecord_Output
	err = c.Post(ctx, "com.atproto.repo.createRecord", body, out)
	if err != nil {
		err = errors.New("oops! failed to create a profile: " + err.Error())
		return
	}
	return profile, nil
}
