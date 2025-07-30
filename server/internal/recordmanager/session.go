package recordmanager

import (
	"context"
	"errors"
)

func (rm *RecordManager) DeleteSession(id int, ctx context.Context) error {
	err := rm.db.DeleteOauthSession(id, ctx)
	if err != nil {
		return errors.New("failed to delete session: " + err.Error())
	}
	rm.clientmap.Delete(id)
	return nil
}
