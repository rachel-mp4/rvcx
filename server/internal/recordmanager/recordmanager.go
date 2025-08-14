package recordmanager

import (
	"context"
	"errors"
	"fmt"
	"rvcx/internal/db"
	"rvcx/internal/log"
	"rvcx/internal/oauth"
	"rvcx/internal/types"
)

type LexBroadcaster interface {
	BroadcastSignet(uri string, s *types.Signet) error
	BroadcastMessage(uri string, m *types.Message) error
	AddChannel(c *types.Channel) error
	UpdateChannel(c *types.Channel) error
	DeleteChannel(uri string) error
}

type RecordManager struct {
	log         *log.Logger
	db          *db.Store
	myClient    *oauth.PasswordClient
	clientmap   *oauth.ClientMap
	broadcaster LexBroadcaster
}

func New(log *log.Logger, db *db.Store, myClient *oauth.PasswordClient, service *oauth.Service) *RecordManager {
	clientmap := oauth.NewClientMap(service)
	return &RecordManager{log, db, myClient, clientmap, nil}
}

func (rm *RecordManager) SetBroadcaster(b LexBroadcaster) {
	rm.broadcaster = b
}

func (rm *RecordManager) getClient(id int, ctx context.Context) (*oauth.OauthXRPCClient, error) {
	cli, refreshed, err := rm.clientmap.Map(id, ctx)
	if cli == nil {
		cli, err = rm.resetClient(id, ctx)
		if err != nil {
			return nil, err
		}
		return cli, nil
	}

	if err != nil {
		return nil, errors.New("error getting client: " + err.Error())
	}
	if refreshed {
		rm.db.UpdateSession(id, cli.GetSession(), ctx)
	}

	return cli, nil
}

func (rm *RecordManager) resetClient(id int, ctx context.Context) (*oauth.OauthXRPCClient, error) {
	session, err := rm.db.GetOauthSession(id, ctx)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("errpr setting up session %d: %s", id, err.Error()))
	}
	return rm.setupClient(session), nil
}

func (rm *RecordManager) setupClient(session *types.Session) *oauth.OauthXRPCClient {
	client := oauth.NewOauthXRPCClient(rm.db, rm.log, session)
	rm.clientmap.Append(session.ID, client, session.Expiration)
	return client
}

// create - oauth
// store - db
// broadcast - channels model
