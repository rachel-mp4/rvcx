package recordmanager

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"rvcx/internal/oauth"
)

func (rm *RecordManager) Beep(id string, did string, ctx context.Context) error {
	sdid, err := syntax.ParseDID(did)
	if err != nil {
		return errors.New("aaaa bbeeeebpp : " + err.Error())
	}
	client, err := rm.service.ResumeSession(ctx, sdid, id)
	if err != nil {
		return err
	}
	return oauth.MakeBskyPost(client, "beep_", ctx)
}
