package model

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"net/http"
	"os"
	"rvcx/internal/atputils"
	"rvcx/internal/db"
	"rvcx/internal/lex"
	"rvcx/internal/log"
	"rvcx/internal/oauth"
	"rvcx/internal/types"
	"sync"
	"time"

	"github.com/rachel-mp4/lrcd"
	lrcpb "github.com/rachel-mp4/lrcproto/gen/go"
)

type Model struct {
	store  *db.Store
	uriMap map[string]*channelModel
	logger *log.Logger
	cli    *oauth.PasswordClient
	mu     sync.Mutex
}

type channelModel struct {
	uri         string
	welcome     string
	serverModel *serverModel
	streamModel *lexStreamModel
}

type serverModel struct {
	server   *lrcd.Server
	lastID   uint32
	initChan chan lrcpb.Event_Init
	ctx      context.Context
	cancel   func()
}

type lexStreamModel struct {
	clients    map[*client]bool
	clientsmu  sync.Mutex
	ctx        context.Context
	cancel     func()
	signetBus  chan types.SignetView
	messageBus chan types.MessageView
}

func (m *Model) GetWSHandlerFrom(uri string) (http.HandlerFunc, error) {
	server, err := m.getServer(uri)
	if err != nil {
		return nil, err
	}
	return server.WSHandler(), nil
}

func (m *Model) GetLexStreamFrom(uri string) (http.HandlerFunc, error) {
	lsm, err := m.getLexStream(uri)
	if err != nil {
		return nil, err
	}
	return lsm.WSHandler(uri, m), nil
}

func (m *Model) getLexStream(uri string) (*lexStreamModel, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	cm := m.uriMap[uri]
	if cm == nil {
		return nil, errors.New("Not a valid server")
	}
	if cm.streamModel == nil {
		ctx, cancel := context.WithCancel(context.Background())
		lsm := lexStreamModel{
			clients:    make(map[*client]bool),
			clientsmu:  sync.Mutex{},
			ctx:        ctx,
			cancel:     cancel,
			signetBus:  make(chan types.SignetView, 10),
			messageBus: make(chan types.MessageView, 10),
		}
		cm.streamModel = &lsm
		go lsm.broadcaster()
	}
	return cm.streamModel, nil
}

func Init(store *db.Store, logger *log.Logger, cli *oauth.PasswordClient) *Model {
	uris, err := store.GetChannelURIs(context.Background())
	if err != nil {
		panic(err)
	}
	uriToServerModel := make(map[string]*channelModel, len(uris))
	myid := os.Getenv("MY_IDENTITY")
	for _, uri := range uris {
		valid := (uri.Host == myid)
		beep := channelModel{
			welcome: uri.Topic,
			uri:     uri.URI,
		}
		if valid {
			beep.serverModel = &serverModel{lastID: uri.LastID}
		}
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

func (m *Model) AddChannel(c *types.Channel) error {
	_, ok := m.uriMap[c.URI]
	if ok {
		return errors.New("tried to add existing server!")
	}
	valid := (c.Host == os.Getenv("MY_IDENTITY"))
	var welcome string
	if c.Topic == nil {
		welcome = "and now you're connected"
	} else {
		welcome = *c.Topic
	}
	beep := channelModel{
		welcome: welcome,
		uri:     c.URI,
	}
	if valid {
		beep.serverModel = &serverModel{lastID: 1}
	}
	m.uriMap[c.URI] = &beep
	return nil
}

func (m *Model) getServer(uri string) (*lrcd.Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cm := m.uriMap[uri]
	if cm == nil {
		return nil, errors.New("Not a valid server")
	}
	sm := cm.serverModel
	if sm == nil {
		return nil, errors.New("Not hosted on this backend!")
	}

	if sm.server == nil {
		var err error
		lastID := sm.lastID
		initChan := make(chan lrcpb.Event_Init, 100)

		server, err := lrcd.NewServer(
			lrcd.WithWelcome(cm.welcome),
			lrcd.WithLogging(os.Stdout, true),
			lrcd.WithInitialID(lastID),
			lrcd.WithInitChannel(initChan),
			lrcd.WithServerURIAndSecret(uri, os.Getenv("LRCD_SECRET")),
		)
		if err != nil {
			return nil, errors.New("Error creating server")
		}

		err = server.Start()
		if err != nil {
			return nil, errors.New("Error starting server")
		}

		if sm.cancel != nil {
			m.logger.Println("that's weird, old cancel lying around")
			sm.cancel()
		}

		ctx, cancel := context.WithCancel(context.Background())
		sm.server = server
		sm.initChan = initChan
		sm.cancel = cancel
		sm.ctx = ctx

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
			cm := m.uriMap[uri]
			if cm == nil || cm.serverModel == nil || cm.serverModel.server == nil {
				m.mu.Unlock()
				return
			}
			sm := cm.serverModel

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
				sm.cancel()
				sm.cancel = nil
				m.mu.Unlock()
				return
			}
			m.mu.Unlock()
		case e, ok := <-initChan:
			if !ok {
				m.logger.Println("this is a weird case!")
				return
			}
			signet := lex.SignetRecord{}
			handle := e.Init.ExternalID
			if handle == nil {
				h := ""
				handle = &h
			}
			signet.AuthorHandle = *handle
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
				URI:          recorduri,
				IssuerDID:    atputils.GetMyDid(),
				AuthorHandle: signet.AuthorHandle,
				ChannelURI:   uri,
				MessageID:    *e.Init.Id,
				CID:          cid,
				StartedAt:    nowTime,
			}
			err = m.store.StoreSignet(&sr, context.Background())
			if err != nil {
				m.logger.Println("failed to store signet!" + err.Error())
			}
			m.BroadcastSignet(uri, sr)
		}
	}
}
