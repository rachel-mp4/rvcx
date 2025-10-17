package recordmanager

import (
	"context"
	"errors"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lrcpb "github.com/rachel-mp4/lrcproto/gen/go"
	"rvcx/internal/atputils"
	"rvcx/internal/lex"
	"rvcx/internal/types"
	"time"
)

func (rm *RecordManager) PostSignet(e lrcpb.Event_Init, uri string, ctx context.Context) error {
	lsr, now, err := rm.validateSignet(e, uri)
	if err != nil {
		return errors.New("failed to validate signet: " + err.Error())
	}
	signet, err := rm.createSignet(lsr, now, *e.Init.Id, ctx)
	if err != nil {
		return errors.New("failed to create signet: " + err.Error())
	}
	wasNew, err := rm.storeSignet(signet, ctx)
	if err != nil {
		return errors.New("failed to store signet: " + err.Error())
	}
	if !wasNew {
		return nil
	}
	rm.log.Deprintln("i was new, so i am forwarding")
	err = rm.forwardSignet(signet, uri)
	if err != nil {
		return errors.New("failed to forward signet: " + err.Error())
	}
	return nil
}

func (rm *RecordManager) DeleteSignet(uri string, ctx context.Context) error {
	rkey, err := atputils.RkeyFromUri(uri)
	if err != nil {
		return errors.New("invalid signet uri: " + err.Error())
	}
	_, err = rm.myClient.DeleteXCVRSignet(rkey, ctx)
	if err != nil {
		return errors.New("failed to delete signet record from repo: " + err.Error())
	}
	err = rm.db.DeleteSignet(uri, ctx)
	if err != nil {
		return errors.New("failed to delete signet from database")
	}
	return nil
}

func (rm *RecordManager) AcceptSignet(s *types.Signet, ctx context.Context) error {
	wasNew, err := rm.storeSignet(s, ctx)
	if err != nil {
		return errors.New("failed to store signet: " + err.Error())
	}
	if !wasNew {
		return nil
	}
	rm.log.Deprintln("i was new & originated elsewhere, so i am forwarding")
	return rm.forwardSignet(s, s.ChannelURI)
}

func (rm *RecordManager) AcceptSignetDelete(uri string, ctx context.Context) error {
	return rm.db.DeleteSignet(uri, ctx)
}

func (rm *RecordManager) AcceptSignetUpdate(s *types.Signet, ctx context.Context) error {
	return rm.db.UpdateSignet(s, ctx)
}

func (rm *RecordManager) validateSignet(e lrcpb.Event_Init, uri string) (*lex.SignetRecord, *time.Time, error) {
	signet := lex.SignetRecord{}
	handle := e.Init.ExternalID
	if handle == nil {
		h := ""
		handle = &h
	}
	signet.AuthorHandle = *handle
	if e.Init.Id == nil {
		return nil, nil, errors.New("ID should not be nil")
	}
	lrcid := uint64(*e.Init.Id)
	signet.LRCID = lrcid
	signet.ChannelURI = uri
	now := syntax.DatetimeNow()
	nowTime := now.Time()
	nowString := now.String()
	signet.StartedAt = &nowString
	return &signet, &nowTime, nil
}

func (rm *RecordManager) createSignet(lsr *lex.SignetRecord, now *time.Time, id uint32, ctx context.Context) (*types.Signet, error) {
	cid, recorduri, err := rm.myClient.CreateXCVRSignet(lsr, ctx)
	if err != nil {
		return nil, errors.New("couldn't create signet: " + err.Error())
	}
	if now == nil {
		return nil, errors.New("wasn't provided time")
	}
	sr := types.Signet{
		URI:          recorduri,
		IssuerDID:    atputils.GetMyDid(),
		AuthorHandle: lsr.AuthorHandle,
		ChannelURI:   lsr.ChannelURI,
		MessageID:    id,
		CID:          cid,
		StartedAt:    *now,
	}
	return &sr, nil
}

func (rm *RecordManager) storeSignet(signet *types.Signet, ctx context.Context) (bool, error) {
	return rm.db.StoreSignet(signet, ctx)
}

func (rm *RecordManager) forwardSignet(signet *types.Signet, uri string) error {
	return rm.broadcaster.BroadcastSignet(uri, signet)
}
