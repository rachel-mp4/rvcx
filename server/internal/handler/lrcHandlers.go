package handler
import (
	"net/http"
	"xcvr-backend/internal/model"
	"fmt"
)

func (h *Handler) acceptWebsocket(w http.ResponseWriter, r *http.Request) {
	rkey := r.PathValue("rkey")
	user := r.PathValue("user")
	uri := fmt.Sprintf("at://%s/org.xcvr.feed.channel/%s", user, rkey)
	f, err := model.GetWSHandlerFrom(uri)
	if err != nil {
		http.NotFound(w, r)
		h.logger.Deprintf("couldn't find user %s's server %s", user, rkey)
		h.logger.Println(err.Error())
		return
	}
	f(w, r)
}

func postChannel(w http.ResponseWriter, r *http.Request) {

}

func postMessage(w http.ResponseWriter, r *http.Request) {

}
