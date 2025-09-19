package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	atpclient "github.com/bluesky-social/indigo/atproto/client"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lexutil "github.com/bluesky-social/indigo/lex/util"

	"mime/multipart"
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
	err = c.Post(ctx, "com.atproto.repo.createRecord", body, &out)
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
	err = c.Post(ctx, "com.atproto.repo.createRecord", body, &out)
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
	err = c.Post(ctx, "com.atproto.repo.createRecord", body, &out)
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
	getOut, err := atproto.RepoGetRecord(ctx, c, "", "org.xcvr.actor.profile", c.AccountDID.String(), "self")
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
	err = c.Post(ctx, "com.atproto.repo.createRecord", body, &out)
	if err != nil {
		err = errors.New("oops! failed to create a profile: " + err.Error())
		return
	}
	return profile, nil
}

func UploadBLOB(cs *oauth.ClientSession, file multipart.File, ctx context.Context) (*lexutil.BlobSchema, error) {
	client := cs.APIClient()

	req := atpclient.NewAPIRequest("POST", "com.atproto.repo.uploadBlob", file)
	resp, err := client.Do(ctx, req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("upload failed withy status %d", resp.StatusCode)
	}
	var result lexutil.BlobSchema
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&result)
	if err != nil {
		return nil, errors.New("failed to decode: " + err.Error())
	}
	return &result, nil
}

func CreateXCVRMedia(cs *oauth.ClientSession, imr *lex.MediaRecord, ctx context.Context) (uri string, cid string, err error) {
	c := cs.APIClient()
	body := map[string]any{
		"collection": "org.xcvr.lrc.message",
		"repo":       *c.AccountDID,
		"record":     imr,
	}
	var out atproto.RepoCreateRecord_Output
	err = c.Post(ctx, "com.atproto.repo.createRecord", body, &out)
	if err != nil {
		err = errors.New("oops! failed to create a media: " + err.Error())
		return
	}
	uri = out.Uri
	cid = out.Cid
	return
}
