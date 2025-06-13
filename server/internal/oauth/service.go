package oauth

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
	"net/http"
	"net/url"
	"strings"

	atoauth "github.com/haileyok/atproto-oauth-golang"
	"github.com/haileyok/atproto-oauth-golang/helpers"
	"github.com/lestrrat-go/jwx/v2/jwk"
)

type Service struct {
	oauth *atoauth.Client
	http  *http.Client
	keys  *jwk.Key
}

func NewService(httpClient *http.Client) (*Service, error) {
	key, err := GetJWKS()
	if err != nil {
		return nil, err
	}
	cid := getClientId()
	cbu := getOauthCallback()
	cli, err := atoauth.NewClient(atoauth.ClientArgs{
		ClientJwk:   *key,
		ClientId:    cid,
		RedirectUri: cbu,
	})
	if err != nil {
		return nil, err
	}
	return &Service{
		oauth: cli,
		http:  httpClient,
		keys:  key,
	}, nil
}

type OauthFlowResult struct {
	AuthzEndpoint string
	State         string
	DID           string
	RequestUri    string
}

type OAuthRequest struct {
	ID                  uint
	AuthserverIss       string
	State               string
	Did                 string
	PdsUrl              string
	PkceVerifier        string
	DpopAuthServerNonce string
	DpopPrivKey         string
}

type CallbackParams struct {
	Iss   string
	State string
	Code  string
}

type Session struct {
	OAuthRequest
	DpopPdsNonce        string
	AccessToken         string
	RefreshToken        string
	Expiration          time.Time
}

func (s *Service) StartAuthFlow(ctx context.Context, handle string) (*OAuthRequest, *OauthFlowResult, error) {
	did, err := s.resolveHandle(handle)
	if err != nil {
		return nil, nil, errors.New("error resolving handle:" + err.Error())
	}
	dpopPrivKey, err := helpers.GenerateKey(nil)
	if err != nil {
		return nil, nil, errors.New("error generating key:" + err.Error())
	}
	dpopPrivKeyJson, err := json.Marshal(dpopPrivKey)
	if err != nil {
		return nil, nil, errors.New("error marshaling privkey to json:" + err.Error())
	}
	parResp, metadata, service, err := s.makeOAuthRequest(ctx, did, handle, dpopPrivKey)
	if err != nil {
		return nil, nil, errors.New("error making oauth request:" + err.Error())
	}
	oauthReq := OAuthRequest{
		AuthserverIss:       metadata.Issuer,
		State:               parResp.State,
		Did:                 did,
		PkceVerifier:        parResp.PkceVerifier,
		DpopAuthServerNonce: parResp.DpopAuthserverNonce,
		DpopPrivKey:         string(dpopPrivKeyJson),
		PdsUrl:              service,
	}
	oauthFlowResult := OauthFlowResult{
		AuthzEndpoint: metadata.AuthorizationEndpoint,
		State:         parResp.State,
		DID:           did,
		RequestUri:    parResp.RequestUri,
	}
	return &oauthReq, &oauthFlowResult, nil

}

func (s *Service) makeOAuthRequest(ctx context.Context, did string, handle string, dpop jwk.Key) (resp *atoauth.SendParAuthResponse, meta *atoauth.OauthAuthorizationMetadata, service string, err error) {
	service, err = s.resolveService(ctx, did)
	if err != nil {
		err = errors.New("error resolving service:" + err.Error())
		return 
	}
	authserver, err := s.oauth.ResolvePdsAuthServer(ctx, service)
	if err != nil {
		err = errors.New("error resolving pds service:" + err.Error())
		return
	}
	meta, err = s.oauth.FetchAuthServerMetadata(ctx, service)
	if err != nil {
		err = errors.New("error fetching auth server metadata:" + err.Error())
		return
	}
	resp, err = s.oauth.SendParAuthRequest(ctx, authserver, meta, handle, "atproto transition:generic", dpop)
	if err != nil {
		err = errors.New("error sending PAR auth request:" + err.Error())
	}
	return
}

func (s *Service) resolveService(ctx context.Context, did string) (string, error) {
	type Identity struct {
		Service []struct {
			ID              string `json:"id"`
			Type            string `json:"type"`
			ServiceEndpoint string `json:"serviceEndpoint"`
		} `json:"service"`
	}
	var url string
	if strings.HasPrefix(did, "did:plc:") {
		url = fmt.Sprintf("https://plc.directory/%s", did)
	} else if strings.HasPrefix(did, "did:web:") {
		url = fmt.Sprintf("https://%s/.well-known/did.json", strings.TrimPrefix(did, "did:web:"))
	} else {
		return "", errors.New("did type not supported")
	}
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", errors.New("error crafting request:" + err.Error())
	}
	resp, err := s.http.Do(req)
	if err != nil {
		return "", errors.New("error evaluating request:" + err.Error())
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		return "", errors.New("could not resolve did to service")
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("error reading response body:" + err.Error())
	}
	var identity Identity
	err = json.Unmarshal(b, &identity)
	if err != nil {
		return "", errors.New("error unmarshaling to identity:" + err.Error())
	}
	var service *string
	for _, svc := range identity.Service {
		if svc.ID == "#atproto_pds" {
			service = &svc.ServiceEndpoint
		}
	}
	if service == nil {
		return "", errors.New("could not find atproto_pds service in resolved did's services")
	}
	return *service, nil
}

func (s *Service) resolveHandle(handle string) (string, error) {
	params := url.Values{
		"handle": []string{handle},
	}
	reqUrl := "https://public.api.bsky.app/xrpc/com.atproto.identity.resolveHandle?" + params.Encode()
	resp, err := s.http.Get(reqUrl)
	if err != nil {
		return "", errors.New("error making handle -> did resolution request:" + err.Error())
	}
	defer resp.Body.Close()

	type did struct {
		Did string
	}
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", errors.New("error reading handle -> did resolution response" + err.Error())
	}
	var resDid did
	err = json.Unmarshal(b, &resDid)
	if err != nil {
		return "", errors.New("error unmarshaling resDid:" + err.Error())
	}
	return resDid.Did, nil
}

func (s *Service) OauthCallback(ctx context.Context, oauthRequest *OAuthRequest, params CallbackParams) (*Session, error) {
	jwk, err := helpers.ParseJWKFromBytes([]byte(oauthRequest.DpopPrivKey))
	if err != nil {
		return nil, errors.New("error parsing jwk:" + err.Error())
	}
	initialTokenResp, err := s.oauth.InitialTokenRequest(ctx, params.Code, params.Iss, oauthRequest.PkceVerifier, oauthRequest.DpopAuthServerNonce, jwk)
	if err != nil {
		return nil, errors.New("error in initialTokenRequest:"+ err.Error())
	}
	if initialTokenResp.Scope != "atproto transition:generic" {
		return nil, errors.New(fmt.Sprintf("incorrect scope: %s", initialTokenResp.Scope))
	}
	oauthSession := Session{
		OAuthRequest: *oauthRequest, 
		AccessToken: initialTokenResp.AccessToken, 
		RefreshToken: initialTokenResp.RefreshToken, 
		Expiration: time.Now().Add(time.Duration(int(time.Second) * int(initialTokenResp.ExpiresIn))),
	}
	return &oauthSession, nil
}
