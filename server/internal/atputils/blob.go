package atputils

import (
	"context"
	"net/http"

	"github.com/bluesky-social/indigo/api/atproto"
	"github.com/bluesky-social/indigo/atproto/client"
)

func SyncGetBlob(did string, cid string, ctx context.Context) ([]byte, error) {
	host, err := GetPDSFromDid(ctx, did, http.DefaultClient)
	if err != nil {
		return nil, err
	}
	c := client.NewAPIClient(host)
	return atproto.SyncGetBlob(ctx, c, cid, did)
}
