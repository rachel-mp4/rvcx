package model

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"rvcx/internal/atputils"
	"rvcx/internal/lex"
	"rvcx/internal/types"
	"time"

	"github.com/gorilla/websocket"
)

type client struct {
	conn *websocket.Conn
	bus  chan any
}

func (cm *channelModel) WSHandler(uri string, m *Model) http.HandlerFunc {
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
		cm.clientsmu.Lock()
		cm.clients[client] = true
		cm.clientsmu.Unlock()

		client.wsWriter()
		cm.logger.Deprintln("i am a lex stream wshandler and i am exiting")

		cm.clientsmu.Lock()
		delete(cm.clients, client)
		cm.clientsmu.Unlock()
	}
}

func (c *client) wsWriter() {
	ticker := time.NewTicker(15 * time.Second)
	for {
		select {
		case <-ticker.C:
			err := c.conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
			if err != nil {
				return
			}
		case e, ok := <-c.bus:
			if !ok {
				return
			}
			c.conn.WriteJSON(e)
		}
	}
}

// func (cm *channelModel) cleanUp() {
// 	cm.clientsmu.Lock()
// 	defer cm.clientsmu.Unlock()
// 	for cli := range cm.clients {
// 		close(cli.bus)
// 	}
// }

func (cm *channelModel) broadcast(a any) {
	cm.clientsmu.Lock()
	defer cm.clientsmu.Unlock()
	for cli := range cm.clients {
		select {
		case cli.bus <- a:
		default:
			delete(cm.clients, cli)
			close(cli.bus)
		}
	}
}

func (m *Model) BroadcastSignet(uri string, s *types.Signet) error {
	cm := m.uriMap[uri]
	if cm == nil {
		return errors.New("AAAAAAAAAAA")
	}
	ihandle, err := m.store.ResolveDid(s.IssuerDID, context.Background())
	if err != nil {
		ihandle, err = atputils.TryLookupDid(context.Background(), s.IssuerDID)
		if err != nil {
			return errors.New("AAAAAAAAAAAAAAAAAAAAA")
		}
		go m.store.StoreDidHandle(s.IssuerDID, ihandle, context.Background())
	}
	sv := types.SignetView{
		URI:          s.URI,
		IssuerHandle: ihandle,
		ChannelURI:   s.ChannelURI,
		LrcId:        s.MessageID,
		AuthorHandle: s.AuthorHandle,
		StartedAt:    s.StartedAt,
	}
	cm.broadcast(sv)
	return nil
}

func (m *Model) BroadcastMessage(uri string, msg *types.Message) error {
	cm := m.uriMap[uri]
	if cm == nil {
		return errors.New("failed to map uri to lsm!")
	}
	pv, err := m.store.GetProfileView(msg.DID, context.Background())
	if err != nil {
		return errors.New("failed to get profile view: " + err.Error())
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
	cm.broadcast(mv)
	return nil
}

func (m *Model) BroadcastImage(uri string, media *types.Image) error {
	cm := m.uriMap[uri]
	if cm == nil {
		return errors.New("failed to map uri to lsm!")
	}
	pv, err := m.store.GetProfileView(media.DID, context.Background())
	if err != nil {
		return errors.New("failed to get profile view: " + err.Error())
	}
	var ar *lex.AspectRatio
	if media.Width != nil && media.Height != nil {
		ar = &lex.AspectRatio{
			Width:  *media.Width,
			Height: *media.Height,
		}
	}
	src := fmt.Sprintf("https://%s/xrpc/org.xcvr.lrc.getImage?uri=%s", os.Getenv("MY_IDENTITY"), media.URI)

	img := types.ImageView{
		Alt:         media.Alt,
		Src:         &src,
		AspectRatio: ar,
	}
	mv := types.MediaView{
		URI:       media.URI,
		Author:    *pv,
		Image:     &img,
		Nick:      media.Nick,
		Color:     media.Color,
		SignetURI: media.SignetURI,
		PostedAt:  media.PostedAt,
	}
	cm.broadcast(mv)
	return nil
}
