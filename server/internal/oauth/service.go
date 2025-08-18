package oauth

import (
	"context"
	"net/url"
	"rvcx/internal/db"

	"github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

type Service struct {
	app *oauth.ClientApp
}

func NewService(store db.Store) (*Service, error) {
	config := oauth.NewPublicConfig(getClientId(), getOauthCallback(), []string{"atproto", "transition:generic"})
	key, err := GetPrivateKey()
	if err != nil {
		return nil, err
	}
	err = config.SetClientSecret(key, "secret.key.name_")
	app := oauth.NewClientApp(&config, store)
	return &Service{app}, nil
}

func (s *Service) StartAuthFlow(ctx context.Context, identifier string) (redirectURL string, err error) {
	return s.app.StartAuthFlow(ctx, identifier)
}

func (s *Service) OauthCallback(ctx context.Context, params url.Values) (sessdata *oauth.ClientSessionData, err error) {
	return s.app.ProcessCallback(ctx, params)
}

func (s *Service) ResumeSession(ctx context.Context, did syntax.DID, sessionId string) (sess *oauth.ClientSession, err error) {
	return s.app.ResumeSession(ctx, did, sessionId)
}
