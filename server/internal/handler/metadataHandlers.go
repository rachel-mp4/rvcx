package handler
import (
	"fmt"
	"net/http"
	"encoding/json"
	"os"
	"xcvr-backend/internal/oauth"
)

func (h *Handler) serveClientMetadata(w http.ResponseWriter, r *http.Request) {
	metadata := oauth.GetClientMetadata()
	encoder := json.NewEncoder(w)
	encoder.Encode(metadata)
}

func (h *Handler) serveTOS(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "be normal be normal be normal be normal be normal be normal be normal")
}
func (h *Handler) servePolicy(w http.ResponseWriter, r *http.Request) {
	fmt.Fprint(w, "i'll be normal i'll be normal i'll be normal i'll be normal")
}

func clientMetadataPath() string {
	mp := os.Getenv("MY_METADATA_PATH")
	return fmt.Sprintf("GET %s", mp)
}

func clientTOSPath() string {
	mp := os.Getenv("MY_TOS_PATH")
	return fmt.Sprintf("GET %s", mp)
}

func clientPolicyPath() string {
	mp := os.Getenv("MY_POLICY_PATH")
	return fmt.Sprintf("GET %s", mp)
}
