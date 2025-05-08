package main

import (
	"net/http"
	"os"
	"context"
	"xcvr-backend/internal/db"
	"xcvr-backend/internal/handler"
	"xcvr-backend/internal/log"

	"github.com/joho/godotenv"
)


func main() {
	logger := log.New(os.Stdout, true)

	gdeerr := godotenv.Load("../.env")
	if gdeerr != nil {
		logger.Println("i think you should make a .env file in the xcvr-backend directory !\n\nExample contents:\n-------------------------------------------------------------------\nPOSTGRES_USER=xcvr\nPOSTGRES_PASSWORD=secret\nPOSTGRES_DB=xcvrdb\nPOSTGRES_PORT=15432\n-------------------------------------------------------------------\n\nGood luck !\n\n")
		panic(gdeerr)	
	}
	conn, err := db.Init()
	defer conn.Close(context.Background())
	if err != nil {
		logger.Println("failed to init db")
		panic(err)
	}
	h := handler.New(conn, logger)
	http.ListenAndServe(":8080", h.WithCORSAll())
	
}

// func initChannel(w http.ResponseWriter, r *http.Request) {
// 	decoder := json.NewDecoder(r.Body)
// 	var c channel
// 	err := decoder.Decode(&c)
// 	if err != nil {
// 		http.Error(w, "invalid json", http.StatusBadRequest)
// 	}
// 	switch isValidInit(c) {
// 	case ieNoBand:
// 		http.Error(w, "must give a band", http.StatusBadRequest)
// 		return
// 	case ieLongBand:
// 		http.Error(w, "band must be shorter than 32 bytes", http.StatusBadRequest)
// 		return
// 	case ieCollision:
// 		http.Error(w, "band must be unique", http.StatusBadRequest)
// 		return
// 	case ieLongSign:
// 		http.Error(w, "sign must be shorter than 51 code points", http.StatusBadRequest)
// 		return
// 	case ieOK:
// 		c, err = createChannel(c, false)
// 	}
// 	if err != nil {
// 		http.Error(w, "uh oh", http.StatusTeapot)
// 	}
// 	fmt.Printf("created a channel on band: %s and call sign: %s\n", c.Band, c.Sign)
// 	encoder := json.NewEncoder(w)
// 	encoder.Encode(c)
// }

// type initError = int

// const (
// 	ieOK initError = iota
// 	ieNoBand
// 	ieLongBand
// 	ieCollision
// 	ieLongSign
// )

// // TODO: can changes to bandToServer after unlock create data race?
// func isValidInit(c channel) initError {
// 	if c.Band == "" {
// 		return ieNoBand
// 	}
// 	if len(c.Band) > 31 {
// 		return ieLongBand
// 	}
// 	channelsMu.Lock()
// 	_, ok := bandToServer[c.Band]
// 	channelsMu.Unlock()
// 	if ok {
// 		return ieCollision
// 	}
// 	if utf8.RuneCountInString(c.Sign) > 50 {
// 		return ieLongSign
// 	}
// 	return ieOK
// }


