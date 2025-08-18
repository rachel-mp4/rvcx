package oauth

import (
	"context"
	"net/url"
	"rvcx/internal/db"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"os"
)

type Service struct {
	App *oauth.ClientApp
}

func NewService(store db.Store) (*Service, error) {
	config := oauth.NewPublicConfig(getClientId(), getOauthCallback(), []string{"atproto", "transition:generic"})
	key, err := GetPrivateKey()
	if err != nil {
		return nil, err
	}
	err = config.SetClientSecret(key, os.Getenv("CLIENT_SECRET_KEY_ID"))
	app := oauth.NewClientApp(&config, store)
	return &Service{app}, nil
}

func (s *Service) StartAuthFlow(ctx context.Context, identifier string) (redirectURL string, err error) {
	return s.App.StartAuthFlow(ctx, identifier)
}

func (s *Service) OauthCallback(ctx context.Context, params url.Values) (sessdata *oauth.ClientSessionData, err error) {
	return s.App.ProcessCallback(ctx, params)
}

func (s *Service) ResumeSession(ctx context.Context, did syntax.DID, sessionId string) (sess *oauth.ClientSession, err error) {
	return s.App.ResumeSession(ctx, did, sessionId)
}
