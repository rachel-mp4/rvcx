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
}

type RecordManager struct {
	log         *log.Logger
	db          *db.Store
	myClient    *oauth.PasswordClient
	clientmap   *oauth.ClientMap
	broadcaster LexBroadcaster
}

func New(log *log.Logger, db *db.Store, myClient *oauth.PasswordClient, broadcaster LexBroadcaster) *RecordManager {
	clientmap := oauth.NewClientMap()
	return &RecordManager{log, db, myClient, clientmap, broadcaster}
}

func (rm *RecordManager) getClient(id int, ctx context.Context) (*oauth.OauthXRPCClient, error) {
	client := rm.clientmap.Map(id)
	if client == nil {
		client, err := rm.resetClient(id, ctx)
		if err != nil {
			return nil, err
		}
		return client, nil
	}
	return client, nil
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
