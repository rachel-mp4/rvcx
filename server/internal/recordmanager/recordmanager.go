package recordmanager

import (
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
	service     *oauth.Service
	broadcaster LexBroadcaster
}

func New(log *log.Logger, db *db.Store, myClient *oauth.PasswordClient, service *oauth.Service) *RecordManager {
	return &RecordManager{log, db, myClient, service, nil}
}

func (rm *RecordManager) SetBroadcaster(b LexBroadcaster) {
	rm.broadcaster = b
}
