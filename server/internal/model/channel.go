package model

import (
	"errors"
	"net/http"
	"sync"
	"time"
	"xcvr-backend/internal/types"

	"github.com/jackc/pgx/v5"
	"github.com/rachel-mp4/lrc/lrcd/pkg/lrcd"
)

var (
	channelsMu  sync.Mutex
	channels    = make([]channel, 0)
	uriToServer = make(map[string]*lrcd.Server)
	didToPView = make(map[string]*pView)
)

type pView struct {
	profileView types.ProfileView
	lastUpdated time.Time
}

type channel struct {
	Title     string `json:"title"`
	Topic     string `json:"topic"`
	CreatedAt string `json:"createdAt"`
	Host      string `json:"host"`
}

func GetWSHandlerFrom(uri string, db *pgx.Conn) (http.HandlerFunc, error) {
	server, ok := uriToServer[uri]
	if !ok {
		return nil, errors.New("channel does not exist")
	}
	return server.WSHandler(), nil
}

// func CreateChannel(title string, topic string) error {
// 	c := channel{Title: title, Topic: topic}
// 	_, err := createChannel(c)
// 	return err
// }

// func createChannel(c channel) (channel, error) {
// 	options := []lrcd.Option{
// 		lrcd.WithWelcome(c.Title),
// 		lrcd.WithLogging(os.Stdout, true),
// 	}
// 	ec := make(chan struct{})

// 	server, err := lrcd.NewServer(options...)

// 	if err != nil {
// 		fmt.Println(err.Error())
// 		return channel{}, err
// 	}
// 	fmt.Println("created", c.Title)

// 	err = server.Start()
// 	if err != nil {
// 		fmt.Println(err.Error())
// 		return channel{}, err
// 	}
// 	fmt.Println("started", c.Title)

// 	channelsMu.Lock()
// 	defer channelsMu.Unlock()
// 	uriToServer[c.Band] = server
// 	channels = append(channels, c)
// 	if withDelete {
// 		go func() {
// 			<-ec
// 			channelsMu.Lock()
// 			idx := slices.Index(channels, c)
// 			channels = slices.Delete(channels, idx, idx+1)
// 			err = bandToServer[c.Band].Stop()
// 			if err != nil {
// 				fmt.Println(err.Error())
// 			}
// 			delete(bandToServer, c.Band)
// 			channelsMu.Unlock()
// 			fmt.Println("deleted", c.Band)
// 		}()
// 	}
// 	return c, nil
// }
