package model

import (
	"context"
	"errors"
	"net/http"
	"os"
	"rvcx/internal/db"
	"rvcx/internal/log"
	"rvcx/internal/oauth"
	"rvcx/internal/recordmanager"
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
	rm     *recordmanager.RecordManager
}

type channelModel struct {
	uri    string
	logger *log.Logger
	valid  bool

	welcome       string
	server        *lrcd.Server
	lastID        uint32
	initChan      <-chan lrcd.InitChanMsg
	mediainitChan <-chan lrcd.MediaInitChanMsg
	ctx           context.Context
	cancel        func()

	clients   map[*client]bool
	clientsmu sync.Mutex
}

func (m *Model) GetWSHandlerFrom(uri string) (http.HandlerFunc, error) {
	server, err := m.getServer(uri)
	if err != nil {
		return nil, err
	}
	return server.WSHandler(), nil
}

func (m *Model) GetLexStreamFrom(uri string) (http.HandlerFunc, error) {
	cm, ok := m.uriMap[uri]
	if !ok {
		return nil, errors.New("not a valid server")
	}
	return cm.WSHandler(uri, m), nil
}

func Init(store *db.Store, logger *log.Logger, cli *oauth.PasswordClient, rm *recordmanager.RecordManager) *Model {
	uris, err := store.GetChannelURIs(context.Background())
	if err != nil {
		panic(err)
	}
	uriToServerModel := make(map[string]*channelModel, len(uris))
	myid := os.Getenv("MY_DID")
	for _, uri := range uris {
		valid := (uri.Host == myid)
		beep := channelModel{
			welcome:   uri.Topic,
			uri:       uri.URI,
			logger:    logger,
			lastID:    uri.LastID,
			valid:     valid,
			clients:   make(map[*client]bool),
			clientsmu: sync.Mutex{},
		}
		uriToServerModel[uri.URI] = &beep
	}
	return &Model{
		store,
		uriToServerModel,
		logger,
		cli,
		sync.Mutex{},
		rm,
	}
}

func (m *Model) AddChannel(c *types.Channel) error {
	_, ok := m.uriMap[c.URI]
	if ok {
		return errors.New("tried to add existing server!")
	}
	valid := (c.Host == os.Getenv("MY_DID"))
	var welcome string
	if c.Topic == nil {
		welcome = "and now you're connected"
	} else {
		welcome = *c.Topic
	}
	beep := channelModel{
		welcome:   welcome,
		uri:       c.URI,
		logger:    m.logger,
		lastID:    1,
		valid:     valid,
		clients:   make(map[*client]bool),
		clientsmu: sync.Mutex{},
	}
	m.uriMap[c.URI] = &beep
	return nil
}

func (m *Model) UpdateChannel(c *types.Channel) error {
	cm, ok := m.uriMap[c.URI]
	if !ok {
		return m.AddChannel(c)
	}
	valid := (c.Host == os.Getenv("my_IDENTITY"))
	if valid != cm.valid {
		if valid {
			cm.valid = true
		} else {
			cm.valid = false
			cm.cancel()
		}
	}
	var welcome string
	if c.Topic == nil {
		welcome = "and now you're connected"
	} else {
		welcome = *c.Topic
	}
	cm.welcome = welcome
	return nil
}

func (m *Model) DeleteChannel(uri string) error {
	cm, ok := m.uriMap[uri]
	if !ok {
		return nil
	}
	delete(m.uriMap, uri)
	// this case is for if a malformed channel record is ingested which
	// doesn't create a channel, but it still shows up in uriMap. probs
	// shouldn't be in uriMap but idk
	if cm != nil {
		cm.cancel()
	}
	return nil
}

func (m *Model) getServer(uri string) (*lrcd.Server, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	cm := m.uriMap[uri]
	if cm == nil {
		return nil, errors.New("uri doesn't refer to a channel i am aware of")
	}
	if !cm.valid {
		return nil, errors.New("Not hosted on this backend!")
	}

	if cm.server == nil {
		m.logger.Deprintln("i think the server should exist, so i'm making it")
		var err error
		lastID := cm.lastID
		initChan := make(chan lrcd.InitChanMsg, 100)
		mediainitChan := make(chan lrcd.MediaInitChanMsg, 100)

		server, err := lrcd.NewServer(
			lrcd.WithResolver(func(externalID string, ctx context.Context) *string {
				did, err := m.store.FullResolveHandle(externalID, ctx)
				if err != nil {
					select {
					case <-ctx.Done():
						return nil
					default:
						return &did
					}
				}
				return &did
			}),
			lrcd.WithWelcome(cm.welcome),
			lrcd.WithLogging(os.Stdout, true),
			lrcd.WithInitialID(lastID),
			lrcd.WithInitChannel(initChan),
			lrcd.WithMediainitChannel(mediainitChan),
			lrcd.WithServerURIAndSecret(uri, os.Getenv("LRCD_SECRET")),
		)
		if err != nil {
			return nil, errors.New("Error creating server")
		}

		err = server.Start()
		if err != nil {
			return nil, errors.New("Error starting server")
		}

		if cm.cancel != nil {
			m.logger.Println("that's weird, old cancel lying around")
			cm.cancel()
		}

		ctx, cancel := context.WithCancel(context.Background())
		cm.server = server
		cm.initChan = initChan
		cm.mediainitChan = mediainitChan
		cm.cancel = cancel
		cm.ctx = ctx

		go m.handleInitEvents(cm)
	}
	return cm.server, nil
}

func (m *Model) handleInitEvents(cm *channelModel) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-cm.ctx.Done():
			cm.logger.Deprintln("i'm a handleinitevent goroutine and my context is done")
			return
		case <-ticker.C:
			c := cm.server.Connected()
			if c == 0 {
				cm.logger.Deprintln("i think the server is empty! gonna break some things")
				lastID, err := cm.server.Stop()
				if err != nil {
					panic(err)
				}
				cm.lastID = lastID
				cm.server = nil
				cm.initChan = nil
				cm.cancel()
				cm.cancel = nil
				return
			}
		case e, ok := <-cm.initChan:
			if !ok {
				cm.logger.Println("this is a weird case!")
				return
			}
			err := m.rm.PostSignet(e.ResolvedId, e.Init, cm.uri, context.Background())
			if err != nil {
				m.logger.Println("error posting signet: " + err.Error())
			}
		case me, ok := <-cm.mediainitChan:
			if !ok {
				cm.logger.Println("this is a weird case!")
				return
			}
			e := lrcpb.Event_Init{
				Init: &lrcpb.Init{
					Id:         me.Mediainit.Mediainit.Id,
					ExternalID: me.Mediainit.Mediainit.ExternalID,
				},
			}
			err := m.rm.PostSignet(me.ResolvedId, e, cm.uri, context.Background())
			if err != nil {
				m.logger.Println("error posting signet: " + err.Error())
			}
		}
	}
}
