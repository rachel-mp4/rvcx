package recordmanager

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"rvcx/internal/atputils"
	"rvcx/internal/lex"
	"rvcx/internal/types"
	"time"
)

func (rm *RecordManager) AcceptChannel(c *types.Channel, ctx context.Context) error {
	err := rm.storeChannel(c, ctx)
	if err != nil {
		return errors.New("failed to store channel: " + err.Error())
	}
	err = rm.initChannel(c)
	if err != nil {
		return errors.New("failed to initialize channel: " + err.Error())
	}
	return nil
}

func (rm *RecordManager) AcceptChannelUpdate(c *types.Channel, ctx context.Context) error {
	err := rm.updateChanneldb(c, ctx)
	if err != nil {
		return errors.New("failed to update channel: " + err.Error())
	}
	err = rm.updateChannelmodel(c)
	if err != nil {
		return errors.New("failed to update channel model: " + err.Error())
	}
	return nil
}

func (rm *RecordManager) AcceptChannelDelete(uri string, ctx context.Context) error {
	err := rm.db.DeleteChannel(uri, ctx)
	if err != nil {
		return errors.New("failed to delete channel: " + err.Error())
	}
	return rm.broadcaster.DeleteChannel(uri)
}

func (rm *RecordManager) PostMyChannel(ctx context.Context, pcr *types.PostChannelRequest) (did string, uri string, err error) {
	return rm.postchannelflow(rm.createMyChannel(), ctx, pcr)
}

func (rm *RecordManager) PostChannel(id int, udid string, ctx context.Context, pcr *types.PostChannelRequest) (did string, uri string, err error) {
	return rm.postchannelflow(rm.createChannel(id, udid), ctx, pcr)
}

func (rm *RecordManager) postchannelflow(f func(*lex.ChannelRecord, *time.Time, context.Context) (*types.Channel, error), ctx context.Context, pcr *types.PostChannelRequest) (did string, uri string, err error) {
	lcr, now, err := rm.validateChannel(pcr)
	if err != nil {
		err = errors.New("couldn't validate channel: " + err.Error())
		return
	}
	channel, err := f(lcr, now, ctx)
	if err != nil {
		err = errors.New("couldn't create channel: " + err.Error())
		return
	}
	err = rm.storeChannel(channel, ctx)
	if err != nil {
		err = errors.New("couldn't store channel: " + err.Error())
		return
	}
	err = rm.initChannel(channel)
	if err != nil {
		err = errors.New("couldn't init channel: " + err.Error())
		return
	}
	did = channel.DID
	uri = channel.URI
	return
}

func (rm *RecordManager) storeChannel(c *types.Channel, ctx context.Context) error {
	return rm.db.StoreChannel(c, ctx)
}

func (rm *RecordManager) initChannel(c *types.Channel) error {
	return rm.broadcaster.AddChannel(c)
}

func (rm *RecordManager) updateChanneldb(c *types.Channel, ctx context.Context) error {
	return rm.db.UpdateChannel(c, ctx)
}

func (rm *RecordManager) updateChannelmodel(c *types.Channel) error {
	return rm.broadcaster.UpdateChannel(c)
}

func (rm *RecordManager) createChannel(id int, did string) func(*lex.ChannelRecord, *time.Time, context.Context) (*types.Channel, error) {
	return func(lcr *lex.ChannelRecord, now *time.Time, ctx context.Context) (*types.Channel, error) {
		client, err := rm.getClient(id, ctx)
		if err != nil {
			return nil, errors.New("couldn't get client")
		}
		uri, cid, err := client.CreateXCVRChannel(lcr, ctx)
		if err != nil {
			return nil, errors.New("something bad probs happened when posting a channel " + err.Error())
		}
		channel := types.Channel{
			URI:       uri,
			CID:       cid,
			DID:       did,
			Host:      lcr.Host,
			Title:     lcr.Title,
			Topic:     lcr.Topic,
			CreatedAt: *now,
			IndexedAt: time.Now(),
		}
		return &channel, nil
	}
}

func (rm *RecordManager) createMyChannel() func(*lex.ChannelRecord, *time.Time, context.Context) (*types.Channel, error) {
	return func(lcr *lex.ChannelRecord, now *time.Time, ctx context.Context) (*types.Channel, error) {
		cid, uri, err := rm.myClient.CreateXCVRChannel(lcr, ctx)
		if err != nil {
			return nil, errors.New("something bad probs happened when posting a channel " + err.Error())
		}
		channel := types.Channel{
			URI:       uri,
			CID:       cid,
			DID:       atputils.GetMyDid(),
			Host:      lcr.Host,
			Title:     lcr.Title,
			Topic:     lcr.Topic,
			CreatedAt: *now,
			IndexedAt: time.Now(),
		}
		return &channel, nil
	}
}

func (rm *RecordManager) validateChannel(cr *types.PostChannelRequest) (*lex.ChannelRecord, *time.Time, error) {
	var lcr lex.ChannelRecord
	if cr.Title == "" || atputils.ValidateGraphemesAndLength(cr.Title, 64, 640) {
		return nil, nil, errors.New("title empty or too long")
	}
	lcr.Title = cr.Title
	if cr.Host == "" {
		return nil, nil, errors.New("no host")
	}
	lcr.Host = cr.Host
	if cr.Topic != nil {
		if atputils.ValidateGraphemesAndLength(*cr.Topic, 256, 2560) {
			return nil, nil, errors.New("topic too long")
		}
		lcr.Topic = cr.Topic
	}

	dtn := syntax.DatetimeNow()
	lcr.CreatedAt = dtn.String()
	time := dtn.Time()
	return &lcr, &time, nil
}
