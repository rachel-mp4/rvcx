package types

import (
	"time"
)

type OauthFlowResult struct {
	AuthzEndpoint string
	State         string
	DID           string
	RequestUri    string
}

type OAuthRequest struct {
	ID                  int
	AuthserverIss       string
	State               string
	Did                 string
	PdsUrl              string
	PkceVerifier        string
	DpopAuthServerNonce string
	DpopPrivKey         string
}

type Session struct {
	OAuthRequest
	DpopPdsNonce string
	AccessToken  string
	RefreshToken string
	Expiration   time.Time
}

type Ban struct {
	Id       int        `json:"id"`
	Did      string     `json:"did"`
	Reason   *string    `json:"reason,omitempty"`
	Till     *time.Time `json:"till,omitempty"`
	BannedAt time.Time  `json:"bannedAt"`
}
