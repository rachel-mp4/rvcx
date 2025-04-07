package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/rachel-mp4/lrc/lrcd/pkg/lrcd"
)

func main() {
	fmt.Println("hello world")
	server, err := lrcd.NewServer(lrcd.WithWSPort(8080), lrcd.WithLogging(os.Stdout, true))
	err = server.Start()
	if err != nil {
		fmt.Printf(err.Error())
	}
	var wg sync.WaitGroup
	wg.Add(1)
	wg.Wait()
	return
}