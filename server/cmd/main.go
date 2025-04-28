package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"sync"
	"time"
	"unicode/utf8"

	"github.com/rachel-mp4/lrc/lrcd/pkg/lrcd"
)

var (
	channelsMu   sync.Mutex
	bandToServer map[string]*lrcd.Server
	channels     []channel
)

type channel struct {
	Band string `json:"band"`
	Sign string `json:"sign"`
}

func main() {
	bandToServer = make(map[string]*lrcd.Server)
	channels = make([]channel, 0)
	createChannel(channel{Band: "general", Sign: "hi"}, false)
	createChannel(channel{Band: "sneep", Sign: "snirp"}, false)
	fmt.Println("hello world")
	mux := http.NewServeMux()
	mux.HandleFunc("GET /{band}/ws", acceptWebsocket)
	mux.HandleFunc("GET /xrpc/getChannels", getChannels)
	mux.HandleFunc("POST /xrpc/initChannel", initChannel)

	http.ListenAndServe(":8080", withCORSAll(mux))
}

func acceptWebsocket(w http.ResponseWriter, r *http.Request) {
	band := r.PathValue("band")
	server, ok := bandToServer[band]
	if !ok {
		http.NotFound(w, r)
		return
	}
	h := server.WSHandler()
	h(w, r)
}

func initChannel(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var c channel
	err := decoder.Decode(&c)
	if err != nil {
		http.Error(w, "invalid json", http.StatusBadRequest)
	}
	switch isValidInit(c) {
	case ieNoBand:
		http.Error(w, "must give a band", http.StatusBadRequest)
		return
	case ieLongBand:
		http.Error(w, "band must be shorter than 32 bytes", http.StatusBadRequest)
		return
	case ieCollision:
		http.Error(w, "band must be unique", http.StatusBadRequest)
		return
	case ieLongSign:
		http.Error(w, "sign must be shorter than 51 code points", http.StatusBadRequest)
		return
	case ieOK:
		c, err = createChannel(c, false)
	}
	if err != nil {
		http.Error(w, "uh oh", http.StatusTeapot)
	}
	fmt.Printf("created a channel on band: %s and call sign: %s\n", c.Band, c.Sign)
	encoder := json.NewEncoder(w)
	encoder.Encode(c)
}

type initError = int

const (
	ieOK initError = iota
	ieNoBand
	ieLongBand
	ieCollision
	ieLongSign
)

// TODO: can changes to bandToServer after unlock create data race?
func isValidInit(c channel) initError {
	if c.Band == "" {
		return ieNoBand
	}
	if len(c.Band) > 31 {
		return ieLongBand
	}
	channelsMu.Lock()
	_, ok := bandToServer[c.Band]
	channelsMu.Unlock()
	if ok {
		return ieCollision
	}
	if utf8.RuneCountInString(c.Sign) > 50 {
		return ieLongSign
	}
	return ieOK
}

func getChannels(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)
	err := encoder.Encode(channels)
	if err != nil {
		panic(err)
	}
}

func createChannel(c channel, withDelete bool) (channel, error) {
	options := []lrcd.Option{
		lrcd.WithWelcome(c.Sign),
		lrcd.WithLogging(os.Stdout, true),
	}
	ec := make(chan struct{})

	if withDelete {
		options = append(options, lrcd.WithEmptyChannel(ec))
		after := 10 * time.Second
		options = append(options, lrcd.WithEmptySignalAfter(after))
	}
	server, err := lrcd.NewServer(options...)

	if err != nil {
		fmt.Println(err.Error())
		return channel{}, err
	}
	fmt.Println("created", c.Band)

	err = server.Start()
	if err != nil {
		fmt.Println(err.Error())
		return channel{}, err
	}
	fmt.Println("started", c.Band)

	channelsMu.Lock()
	defer channelsMu.Unlock()
	bandToServer[c.Band] = server
	channels = append(channels, c)
	if withDelete {
		go func() {
			<-ec
			channelsMu.Lock()
			idx := slices.Index(channels, c)
			channels = slices.Delete(channels, idx, idx+1)
			err = bandToServer[c.Band].Stop()
			if err != nil {
				fmt.Println(err.Error())
			}
			delete(bandToServer, c.Band)
			channelsMu.Unlock()
			fmt.Println("deleted", c.Band)
		}()
	}
	return c, nil
}

func withCORSAll(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("incoming request:", r.Method, r.URL.Path)
		w.Header().Set("Access-Control-Allow-Origin", "http://localhost:5173")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		h.ServeHTTP(w, r)
	})
}
