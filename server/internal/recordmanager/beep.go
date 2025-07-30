package recordmanager

import (
	"context"
)

func (rm *RecordManager) Beep(id int, ctx context.Context) error {

	client, err := rm.getClient(id, ctx)
	if err != nil {
		return err
	}
	return client.MakeBskyPost("beep_", ctx)
}
