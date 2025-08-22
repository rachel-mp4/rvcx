package recordmanager

import (
	"context"
	atoauth "github.com/bluesky-social/indigo/atproto/auth/oauth"
	"rvcx/internal/oauth"
)

func (rm *RecordManager) Beep(cs *atoauth.ClientSession, ctx context.Context) error {
	return oauth.MakeBskyPost(cs, "beep_", ctx)
}

func (rm *RecordManager) Unfollow(cs *atoauth.ClientSession, followuri string, ctx context.Context) error {
	return oauth.Unfollow(cs, followuri, ctx)
}
func (rm *RecordManager) Follow(cs *atoauth.ClientSession, did string, ctx context.Context) error {
	return oauth.Follow(cs, did, ctx)
}
