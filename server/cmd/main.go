package main

import (
	"fmt"
	"net/http"

	"github.com/rachel-mp4/lrc/lrcd/pkg/lrcd"
)

var (
	channelToServer map[string]*lrcd.Server
)

func main() {
	channelToServer = make( map[string]*lrcd.Server)
	fmt.Println("hello world")
	http.HandleFunc("GET /", homeHandler)

	http.HandleFunc("GET /{channel}", serverStart)
	http.ListenAndServe(":8080", nil)
}

func serverStart(w http.ResponseWriter, r *http.Request) {
	name := r.PathValue("channel")
	fmt.Fprintln(w,name)
	_, ok := channelToServer[name]
	if ok {
		fmt.Fprint(w, "already created server")
		return
	}
	channelToServer[name] = nil
	fmt.Fprintln(w, "created server")
	// if server != nil {
	// 	fmt.Fprint(w, "server already started")
	// 	return
	// }
	// var err error
	// server, err = lrcd.NewServer(lrcd.WithWSPort(8080), lrcd.WithLogging(os.Stdout, true))
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	fmt.Fprintln(w, "failed to start")
	// 	return
	// }
	// err = server.Start()
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	fmt.Fprintln(w, "failed to start")
	// 	return
	// }
	// fmt.Fprintln(w, "started")
}

func serverStop(w http.ResponseWriter, r *http.Request) {
	// if server == nil {
	// 	fmt.Fprintln(w, "no server to stop")
	// 	return
	// }
	// err := server.Stop()
	// if err != nil {
	// 	fmt.Println(err.Error())
	// 	fmt.Fprintln(w, "failed to stop")
	// 	return
	// }
	// server = nil
	// fmt.Fprintln(w, "stopped")
}

func homeHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintln(w, "welcome")
	for k, _ := range channelToServer {
		fmt.Fprintln(w, k)
	}
}



















