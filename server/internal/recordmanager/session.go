package recordmanager

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
)

func (rm *RecordManager) DeleteSession(did string, sessionID string, ctx context.Context) error {
	sdid, err := syntax.ParseDID(did)
	if err != nil {
		return errors.New("beep boop :  " + err.Error())
	}
	err = rm.db.DeleteSession(ctx, sdid, sessionID)
	if err != nil {
		return errors.New("failed to delete session: " + err.Error())
	}
	return nil
}
