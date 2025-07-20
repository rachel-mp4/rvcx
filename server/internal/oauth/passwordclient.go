package oauth

import (
	"context"
	"errors"
	"fmt"
	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/client"
	"github.com/bluesky-social/indigo/lex/util"
	"os"
	"xcvr-backend/internal/lex"
	"xcvr-backend/internal/log"
)

type PasswordClient struct {
	logger     *log.Logger
	xrpc       *client.APIClient
	accessjwt  *string
	refreshjwt *string
	did        *string
}

func NewPasswordClient(did string, host string, l *log.Logger) *PasswordClient {
	return &PasswordClient{
		xrpc:   client.NewAPIClient(host),
		did:    &did,
		logger: l,
	}
}

func (c *PasswordClient) CreateSession(ctx context.Context) error {
	c.logger.Deprintln("creating session...")
	secret := os.Getenv("MY_SECRET")
	identity := os.Getenv("MY_IDENTITY")
	input := atproto.ServerCreateSession_Input{
		Identifier: identity,
		Password:   secret,
	}
	var out atproto.ServerCreateSession_Output
	err := c.xrpc.LexDo(ctx, "POST", "application/json", "com.atproto.server.createSession", nil, input, &out)
	if err != nil {
		return errors.New("I couldn't create a session: " + err.Error())
	}
	c.accessjwt = &out.AccessJwt
	c.refreshjwt = &out.RefreshJwt
	c.logger.Deprintln("created session!")
	return nil
}

func (c *PasswordClient) RefreshSession(ctx context.Context) error {
	c.logger.Deprintln("refreshing session")
	c.xrpc.Headers.Set("Authorization", fmt.Sprintf("Bearer %s", *c.refreshjwt))
	var out atproto.ServerRefreshSession_Output
	err := c.xrpc.LexDo(ctx, "POST", "application/json", "com.atproto.server.refreshSession", nil, nil, &out)
	if err != nil {
		c.logger.Println("FAILED TO REFRESH RESSION")
		return errors.New("failed to refresh session! " + err.Error())
	}
	c.accessjwt = &out.AccessJwt
	c.refreshjwt = &out.RefreshJwt
	c.logger.Deprintln("refreshed session!")
	return nil
}

func (c *PasswordClient) CreateXCVRMessage(message *lex.MessageRecord, ctx context.Context) (cid string, uri string, err error) {
	input := atproto.RepoCreateRecord_Input{
		Collection: "org.xcvr.lrc.message",
		Repo:       *c.did,
		Record:     &util.LexiconTypeDecoder{Val: message},
	}
	return c.createMyRecord(input, ctx)
}

func (c *PasswordClient) CreateXCVRSignet(signet *lex.SignetRecord, ctx context.Context) (cid string, uri string, err error) {
	input := atproto.RepoCreateRecord_Input{
		Collection: "org.xcvr.lrc.signet",
		Repo:       *c.did,
		Record:     &util.LexiconTypeDecoder{Val: signet},
	}
	return c.createMyRecord(input, ctx)
}

func (c *PasswordClient) CreateXCVRChannel(channel *lex.ChannelRecord, ctx context.Context) (cid string, uri string, err error) {
	input := atproto.RepoCreateRecord_Input{
		Collection: "org.xcvr.lrc.channel",
		Repo:       *c.did,
		Record:     &util.LexiconTypeDecoder{Val: channel},
	}
	return c.createMyRecord(input, ctx)
}

func (c *PasswordClient) createMyRecord(input atproto.RepoCreateRecord_Input, ctx context.Context) (cid string, uri string, err error) {
	if c.accessjwt == nil {
		err = errors.New("must create a session first")
		return
	}
	c.xrpc.Headers.Set("Authorization", fmt.Sprintf("Bearer %s", *c.accessjwt))
	var out atproto.RepoCreateRecord_Output
	err = c.xrpc.LexDo(ctx, "POST", "application/json", "com.atproto.repo.createRecord", nil, input, &out)
	if err != nil {
		err1 := err.Error()
		err = c.RefreshSession(ctx)
		if err != nil {
			err = errors.New(fmt.Sprintf("failed to refresh session while creating %s! first %s then %s", input.Collection, err1, err.Error()))
			return
		}
		c.xrpc.Headers.Set("Authorization", fmt.Sprintf("Bearer %s", *c.accessjwt))
		out = atproto.RepoCreateRecord_Output{}
		err = c.xrpc.LexDo(ctx, "POST", "application/json", "com.atproto.repo.createRecord", nil, input, &out)
		if err != nil {
			err = errors.New(fmt.Sprintf("not good, failed to create %s after failing then refreshing session! first %s then %s", input.Collection, err1, err.Error()))
			return
		}
		cid = out.Cid
		uri = out.Uri
		return
	}
	cid = out.Cid
	uri = out.Uri
	return
}

func (c *PasswordClient) DeleteXCVRSignet(rkey string, ctx context.Context) (bool, error) {
	getOut, err := atproto.RepoGetRecord(ctx, c.xrpc, "", "org.xcvr.lrc.signet", *c.did, rkey)
	if err != nil {
		return true, errors.New("nothing to delete :3")
	}
	input := atproto.RepoDeleteRecord_Input{
		Repo:       *c.did,
		Collection: "org.xcvr.lrc.signet",
		Rkey:       rkey,
		SwapRecord: getOut.Cid,
	}
	err = c.deleteMyRecord(input, ctx)
	if err != nil {
		return false, errors.New("failed to delete")
	}
	return true, nil
}

func (c *PasswordClient) deleteMyRecord(input atproto.RepoDeleteRecord_Input, ctx context.Context) (err error) {
	if c.accessjwt == nil {
		err = errors.New("must create a session first")
		return
	}
	c.xrpc.Headers.Set("Authorization", fmt.Sprintf("Bearer %s", *c.accessjwt))
	var out atproto.RepoDeleteRecord_Output
	err = c.xrpc.LexDo(ctx, "POST", "application/json", "com.atproto.repo.deleteRecord", nil, input, &out)
	if err != nil {
		err1 := err.Error()
		err = c.RefreshSession(ctx)
		if err != nil {
			err = errors.New(fmt.Sprintf("failed to refresh session while deleting %s! first %s then %s", input.Collection, err1, err.Error()))
			return
		}
		c.xrpc.Headers.Set("Authorization", fmt.Sprintf("Bearer %s", *c.accessjwt))
		err = c.xrpc.LexDo(ctx, "POST", "application/json", "com.atproto.repo.deleteRecord", nil, input, &out)
		if err != nil {
			err = errors.New(fmt.Sprintf("not good, failed to delete %s after failing then refreshing session! first %s then %s", input.Collection, err1, err.Error()))
			return
		}
	}
	return
}
