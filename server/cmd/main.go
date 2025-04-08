package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/rachel-mp4/lrc/lrcd/pkg/lrcd"
)

var (
	channelToServer map[string]*lrcd.Server
	channels []channel
)

func main() {
	channelToServer = make(map[string]*lrcd.Server)
	fmt.Println("hello world")
	http.HandleFunc("GET /xrpc/getChannels", getChannels)

	http.HandleFunc("POST /xrpc/initChannel", initChannel)
	http.ListenAndServe(":8080", nil)
}

func getChannels(w http.ResponseWriter, r *http.Request) {
	encoder := json.NewEncoder(w)
	err := encoder.Encode(channels)
	if err != nil {
		panic(err)
	}
}

type channel struct {
	Band string `json:"band"`
	Sign string `json:"sign"`
}

func initChannel(w http.ResponseWriter, r *http.Request) {
	decoder := json.NewDecoder(r.Body)
	var c channel
	err := decoder.Decode(&c)
	if err != nil {
		panic(err)
	}
	channels = append(channels, c)
	fmt.Printf("created a channel on band: %s and call sign: %s\n", c.Band, c.Sign)
}










