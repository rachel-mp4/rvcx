package recordmanager

import (
	"context"
	atoauth "github.com/bluesky-social/indigo/atproto/auth/oauth"
	"rvcx/internal/oauth"
)

func (rm *RecordManager) Beep(cs *atoauth.ClientSession, ctx context.Context) error {
	return oauth.MakeBskyPost(cs, "beep_", ctx)
}
