package oauth

import (
	"fmt"
	"os"
)

var (
	mi             = os.Getenv("MY_IDENTITY")
	mp             = os.Getenv("MY_METADATA_PATH")
	clientMetadata *ClientMetadata
)

type ClientMetadata struct {
	ClientId                    string   `json:"client_id"`
	ClientName                  string   `json:"client_name"`
	ClientUri                   string   `json:"client_uri"`
	LogoUri                     string   `json:"logo_uri"`
	TosUri                      string   `json:"tos_uri"`
	PolicyUrl                   string   `json:"policy_url"`
	RedirectUris                []string `json:"redirect_uris"`
	GrantTypes                  []string `json:"grant_types"`
	ResponseTypes               []string `json:"response_types"`
	ApplicationType             string   `json:"application_type"`
	DPOPBoundAccessTokens       bool     `json:"dpop_bound_access_tokens"`
	JWKSUri                     string   `json:"jwks_uri"`
	Scope                       string   `json:"scope"`
	TokenEndpointAuthMethod     string   `json:"token_endpoing_auth_method"`
	TokenEndpointAuthSigningAlg string   `json:"token_endpoint_auth_signing_alg"`
}

func GetClientMetadata() ClientMetadata {
	if clientMetadata == nil {
		clientMetadata = &ClientMetadata{
			ClientId:                    getClientId(),
			ClientName:                  getClientName(),
			ClientUri:                   getClientUri(),
			LogoUri:                     getLogoUri(),
			TosUri:                      getTOSUri(),
			PolicyUrl:                   getPolicyUri(),
			RedirectUris:                []string{getOauthCallback()},
			GrantTypes:                  []string{"authorization_code", "refresh_token"},
			ResponseTypes:               []string{"code"},
			ApplicationType:             "web",
			DPOPBoundAccessTokens:       true,
			JWKSUri:                     getJWKSUri(),
			Scope:                       "atproto transition:generic",
			TokenEndpointAuthMethod:     "private_key_jwt",
			TokenEndpointAuthSigningAlg: "ES256",
		}
	}
	return *clientMetadata
}

func getClientId() string {
	return fmt.Sprintf("https://%s%s", mi, mp)
}

func getClientName() string {
	return os.Getenv("MY_NAME")
}

func getClientUri() string {
	return fmt.Sprintf("https://%s", mi)
}

func getLogoUri() string {
	return fmt.Sprintf("%s%s", getClientUri(), os.Getenv("MY_LOGO_PATH"))
}

func getTOSUri() string {
	return fmt.Sprintf("%s%s", getClientUri(), os.Getenv("MY_TOS_PATH"))
}

func getPolicyUri() string {
	return fmt.Sprintf("%s%s", getClientUri(), os.Getenv("MY_POLICY_PATH"))
}

func getOauthCallback() string {
	return fmt.Sprintf("%s%s", getClientUri(), os.Getenv("MY_OAUTH_CALLBACK"))
}

func getJWKSUri() string {
	return fmt.Sprintf("%s%s", getClientUri(), os.Getenv("MY_JWKS_PATH"))
}
