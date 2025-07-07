package model

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/http"
	"os"
	"sync"
	"time"
	"xcvr-backend/internal/atputils"
	"xcvr-backend/internal/db"
	"xcvr-backend/internal/lex"
	"xcvr-backend/internal/log"
	"xcvr-backend/internal/oauth"
	"xcvr-backend/internal/types"

	"github.com/rachel-mp4/lrcd"
	lrcpb "github.com/rachel-mp4/lrcproto/gen/go"
)

type Model struct {
	store  *db.Store
	uriMap map[string]*serverModel
	logger *log.Logger
	cli    *oauth.PasswordClient
	mu     sync.Mutex
}

type serverModel struct {
	valid      bool
	server     *lrcd.Server
	lastID     uint32
	initChan   chan lrcpb.Event_Init
	cancelFunc func()
}

func (m *Model) GetWSHandlerFrom(uri string) (http.HandlerFunc, error) {
	server, err := m.getServer(uri)
	if err != nil {
		return nil, err
	}
	return server.WSHandler(), nil
}

func Init(store *db.Store, logger *log.Logger, cli *oauth.PasswordClient) *Model {
	uris, err := store.GetChannelURIs(context.Background())
	if err != nil {
		panic(err)
	}
	uriToServerModel := make(map[string]*serverModel, len(uris))
	myid := os.Getenv("MY_IDENTITY")
	for _, uri := range uris {
		valid := (uri.Host == myid)
		beep := serverModel{valid: valid}
		uriToServerModel[uri.URI] = &beep
	}
	return &Model{
		store,
		uriToServerModel,
		logger,
		cli,
		sync.Mutex{},
	}
}

func (m *Model) getServer(uri string) (*lrcd.Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sm := m.uriMap[uri]
	if sm == nil {
		return nil, errors.New("Not a valid server")
	}
	if !sm.valid {
		return nil, errors.New("Not hosted on this backend!")
	}

	if sm.server == nil {
		var err error
		lastID := sm.lastID
		initChan := make(chan lrcpb.Event_Init, 100)

		server, err := lrcd.NewServer(
			lrcd.WithLogging(os.Stdout, true),
			lrcd.WithInitialID(lastID),
			lrcd.WithInitChannel(initChan),
		)
		if err != nil {
			return nil, errors.New("Error creating server")
		}

		err = server.Start()
		if err != nil {
			return nil, errors.New("Error starting server")
		}

		if sm.cancelFunc != nil {
			sm.cancelFunc()
		}

		ctx, cancel := context.WithCancel(context.Background())
		sm.server = server
		sm.initChan = initChan
		sm.cancelFunc = cancel

		go m.handleInitEvents(ctx, uri, initChan)
	}
	return sm.server, nil
}

func (m *Model) handleInitEvents(ctx context.Context, uri string, initChan <-chan lrcpb.Event_Init) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.mu.Lock()
			sm := m.uriMap[uri]
			if sm == nil || sm.server == nil {
				m.mu.Unlock()
				return
			}

			c := sm.server.Connected()
			if c == 0 {
				lastID, err := sm.server.Stop()
				if err != nil {
					m.mu.Unlock()
					panic(err)
				}
				sm.lastID = lastID
				sm.server = nil
				sm.initChan = nil
				m.mu.Unlock()
				return
			}
			m.mu.Unlock()
		case e, ok := <-initChan:
			if !ok {
				return
			}
			signet := lex.SignetRecord{}
			handle := e.Init.ExternalID
			if handle == nil {
				h := ""
				handle = &h
			}
			signet.Author = *handle
			if e.Init.Id == nil {
				m.logger.Deprintln("initchannel gave me a nil id")
				continue
			}
			lrcid := uint64(*e.Init.Id)
			signet.LRCID = lrcid
			signet.ChannelURI = uri
			now := syntax.DatetimeNow()
			nowTime := now.Time()
			nowString := now.String()
			signet.StartedAt = &nowString
			cid, recorduri, err := m.cli.CreateXCVRSignet(&signet, context.Background())
			if err != nil {
				m.logger.Deprintf("couldn't post a signet in %s: %s", uri, err.Error())
				continue
			}
			sr := types.Signet{
				URI:        recorduri,
				IssuerDID:  atputils.GetMyDid(),
				DID:        signet.Author,
				ChannelURI: uri,
				MessageID:  *e.Init.Id,
				CID:        cid,
				StartedAt:  nowTime,
			}
			err = m.store.StoreSignet(sr, context.Background())
			if err != nil {
				m.logger.Println("failed to store signet!")
			}
		}
	}
}
