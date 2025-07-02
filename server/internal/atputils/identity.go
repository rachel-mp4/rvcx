package atputils

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bluesky-social/indigo/atproto/identity"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"io"
	"net/http"
	"os"
	"strings"
)

var (
	my_handle *string
	my_did    *string
)

func GetMyHandle() string {
	if my_handle != nil {
		return *my_handle
	}
	handle := os.Getenv("MY_IDENTITY")
	my_handle = &handle
	return *my_handle
}

func GetMyDid(ctx context.Context) (string, error) {
	if my_did != nil {
		return *my_did, nil
	}
	if my_handle == nil {
		GetMyHandle()
	}
	did, err := GetDidFromHandle(ctx, *my_handle)
	if err != nil {
		return "", err
	}
	my_did = &did
	return did, nil
}

func GetHandleFromDid(ctx context.Context, did string) (string, error) {
	sdid, err := syntax.ParseDID(did)
	if err != nil {
		return "", errors.New("did did not parse: " + err.Error())
	}
	resolver := identity.DefaultDirectory()

	ident, err := resolver.LookupDID(ctx, sdid)
	if err != nil {
		return "", errors.New("failed to lookupDID: " + err.Error())
	}
	return ident.Handle.String(), nil
}

func GetDidFromHandle(ctx context.Context, handle string) (string, error) {
	shandle, err := syntax.ParseHandle(handle)
	if err != nil {
		return "", errors.New("handle did not parse: " + err.Error())
	}
	resolver := identity.DefaultDirectory()
	ident, err := resolver.LookupHandle(ctx, shandle)
	if err != nil {
		return "", errors.New("failed to lookupHandle: " + err.Error())
	}
	return ident.DID.String(), nil
}

func GetPDSFromHandle(ctx context.Context, handle string) (string, error) {
	did, err := GetDidFromHandle(ctx, handle)
	if err != nil {
		return "", errors.New("failed to find did from handle in handle->pds: " + err.Error())
	}
	return GetPDSFromDid(ctx, did, http.DefaultClient)
}

func GetPDSFromDid(ctx context.Context, did string, cli *http.Client) (string, error) {
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
	resp, err := cli.Do(req)
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
