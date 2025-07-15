package model

import (
	"context"
	"github.com/gorilla/websocket"
	"net/http"
	"time"
	"xcvr-backend/internal/types"
)

type client struct {
	conn *websocket.Conn
	bus  chan any
}

func (lsm *lexStreamModel) WSHandler(uri string, m *Model) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		upgrader := &websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		}
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		bus := make(chan any, 10)
		client := &client{
			conn,
			bus,
		}
		lsm.clientsmu.Lock()
		lsm.clients[client] = true
		lsm.clientsmu.Unlock()

		client.wsWriter(lsm.ctx)

		lsm.clientsmu.Lock()
		delete(lsm.clients, client)
		if len(lsm.clients) == 0 {
			lsm.cancel()
			m.uriMap[uri].streamModel = nil
		}
		lsm.clientsmu.Unlock()
	}
}

func (c *client) wsWriter(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-ticker.C:
			err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
			if err != nil {
				return
			}
		case <-ctx.Done():
			return
		case e, ok := <-c.bus:
			if !ok {
				return
			}
			c.conn.WriteJSON(e)
		}
	}
}

func (lsm *lexStreamModel) broadcaster() {
	for {
		select {
		case <-lsm.ctx.Done():
			lsm.cleanUp()
			return
		case m, ok := <-lsm.messageBus:
			if !ok {
				lsm.cleanUp()
				return
			}
			lsm.broadcast(m)
		case s, ok := <-lsm.signetBus:
			if !ok {
				lsm.cleanUp()
				return
			}
			lsm.broadcast(s)
		}
	}
}

func (lsm *lexStreamModel) cleanUp() {
	lsm.clientsmu.Lock()
	defer lsm.clientsmu.Unlock()
	for cli := range lsm.clients {
		close(cli.bus)
	}
}

func (lsm *lexStreamModel) broadcast(a any) {
	lsm.clientsmu.Lock()
	defer lsm.clientsmu.Unlock()
	for cli := range lsm.clients {
		select {
		case cli.bus <- a:
		default:
			delete(lsm.clients, cli)
			close(cli.bus)
		}
	}
}

func (m *Model) BroadcastSignet(uri string, s types.Signet) {
	lsm := m.uriMap[uri]
	if lsm == nil {
		return
	}
	ihandle, err := m.store.ResolveDid(s.IssuerDID, context.Background())
	if err != nil {
		return
	}
	sv := types.SignetView{
		URI:          s.URI,
		IssuerHandle: ihandle,
		ChannelURI:   s.ChannelURI,
		LrcId:        s.MessageID,
		AuthorHandle: s.AuthorHandle,
		StartedAt:    s.StartedAt,
	}
	lsm.streamModel.signetBus <- sv
}

func (m *Model) BroadcastMessage(uri string, msg types.Message) {
	lsm := m.uriMap[uri]
	if lsm == nil {
		return
	}
	pv, err := m.store.GetProfileView(msg.DID, context.Background())
	if err != nil {
		return
	}
	mv := types.MessageView{
		URI:       msg.URI,
		Author:    *pv,
		Body:      msg.Body,
		Nick:      msg.Nick,
		Color:     msg.Color,
		SignetURI: msg.SignetURI,
		PostedAt:  msg.PostedAt,
	}
	lsm.streamModel.messageBus <- mv
}
