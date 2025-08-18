package recordmanager

import (
	"context"
	"errors"
	atoauth "github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/rachel-mp4/lrcd"
	"os"
	"rvcx/internal/atputils"
	"rvcx/internal/lex"
	"rvcx/internal/oauth"
	"rvcx/internal/types"
	"slices"
	"time"
)

func (rm *RecordManager) AcceptMessage(m *types.Message, ctx context.Context) error {
	err := rm.storeMessage(m, ctx)
	if err != nil {
		return errors.New("failed to store message: " + err.Error())
	}
	err = rm.forwardMessage(m, ctx)
	if err != nil {
		return errors.New("failed to forward message: " + err.Error())
	}
	return nil
}

func (rm *RecordManager) AcceptMessageUpdate(m *types.Message, did string, ctx context.Context) error {
	err := rm.updateMessage(m, ctx)
	if err != nil {
		return errors.New("failed to store message: " + err.Error())
	}
	err = rm.checkInterference(m, did, ctx)
	if err != nil {
		return errors.New("error while checking interference: " + err.Error())
	}
	return nil
}

func (rm *RecordManager) AcceptMessageDelete(uri string, ctx context.Context) error {
	err := rm.db.DeleteMessage(uri, ctx)
	if err != nil {
		return errors.New("failed to delete message: " + err.Error())
	}
	return nil
}

func (rm *RecordManager) checkInterference(m *types.Message, did string, ctx context.Context) error {
	handle, err := rm.db.QuerySignetHandle(m.SignetURI, ctx)
	if err != nil {
		return errors.New("couldn't find signet")
	}
	sdid, err := atputils.DidFromUri(m.SignetURI)
	if sdid != atputils.GetMyDid() {
		return nil
	}
	mhandle, err := rm.db.ResolveDid(did, ctx)
	if err != nil {
		return errors.New("couldn't resolve mhandle")
	}
	if handle != mhandle {
		return nil
	}
	err = rm.DeleteSignet(m.SignetURI, ctx)
	if err != nil {
		return errors.New("failed to delete signet")
	}
	return nil
}

func (rm *RecordManager) PostMessage(cs *atoauth.ClientSession, ctx context.Context, pmr *types.PostMessageRequest) error {
	rm.log.Deprintln("validate")
	lmr, now, _, _, err := rm.validateMessage(pmr, ctx)
	if err != nil {
		return errors.New("failed to validate message: " + err.Error())
	}
	rm.log.Deprintln("create")
	m, err := rm.createMessage(cs, lmr, now, ctx)
	if err != nil {
		return errors.New("failed to create message: " + err.Error())
	}
	rm.log.Deprintln("store")
	err = rm.storeMessage(m, ctx)
	if err != nil {
		return errors.New("failed to store message: " + err.Error())
	}
	rm.log.Deprintln("forward")
	err = rm.forwardMessage(m, ctx)
	if err != nil {
		return errors.New("failed to forward message: " + err.Error())
	}
	return nil
}

func (rm *RecordManager) PostMyMessage(ctx context.Context, pmr *types.PostMessageRequest) error {
	lmr, now, handle, nonce, err := rm.validateMessage(pmr, ctx)
	if err != nil {
		return errors.New("failed to validate message: " + err.Error())
	}
	err = rm.validateHandleAndNonce(handle, nonce, lmr.SignetURI, ctx)
	if err != nil {
		return errors.New("failed to validate my handle and nonce: " + err.Error())
	}
	m, err := rm.createMyMessage(lmr, now, ctx)
	if err != nil {
		return errors.New("failed to create message: " + err.Error())
	}
	err = rm.storeMessage(m, ctx)
	if err != nil {
		return errors.New("failed to store message: " + err.Error())
	}
	err = rm.forwardMessage(m, ctx)
	if err != nil {
		return errors.New("failed to forward message: " + err.Error())
	}
	return nil
}

func (rm *RecordManager) validateHandleAndNonce(handle *string, nonce []byte, signetUri string, ctx context.Context) error {
	if handle == nil || *handle != atputils.GetMyHandle() {
		return errors.New("i only post my messages")
	}
	curi, mid, err := rm.db.QuerySignetChannelIdNum(signetUri, ctx)
	if err != nil {
		return errors.New("failed to find signet")
	}
	correctNonce := lrcd.GenerateNonce(mid, curi, os.Getenv("LRCD_SECRET"))
	if !slices.Equal(nonce, correctNonce) {
		return errors.New("i think user tried to post someone else's post")
	}
	return nil
}

func (rm *RecordManager) createMyMessage(lmr *lex.MessageRecord, now *time.Time, ctx context.Context) (*types.Message, error) {
	cid, uri, err := rm.myClient.CreateXCVRMessage(lmr, ctx)
	if err != nil {
		return nil, errors.New("couldn't add to user repo: " + err.Error())
	}
	var coloruint32ptr *uint32
	if lmr.Color != nil {
		color := uint32(*lmr.Color)
		coloruint32ptr = &color
	}
	message := &types.Message{
		URI:       uri,
		DID:       atputils.GetMyDid(),
		CID:       cid,
		SignetURI: lmr.SignetURI,
		Body:      lmr.Body,
		Nick:      lmr.Nick,
		Color:     coloruint32ptr,
		PostedAt:  *now,
	}
	return message, nil
}

func (rm *RecordManager) createMessage(cs *atoauth.ClientSession, lmr *lex.MessageRecord, now *time.Time, ctx context.Context) (*types.Message, error) {
	uri, cid, err := oauth.CreateXCVRMessage(cs, lmr, ctx)
	if err != nil {
		return nil, errors.New("couldn't add to user repo: " + err.Error())
	}
	var coloruint32ptr *uint32
	if lmr.Color != nil {
		color := uint32(*lmr.Color)
		coloruint32ptr = &color
	}
	message := &types.Message{
		URI:       uri,
		DID:       cs.Data.AccountDID.String(),
		CID:       cid,
		SignetURI: lmr.SignetURI,
		Body:      lmr.Body,
		Nick:      lmr.Nick,
		Color:     coloruint32ptr,
		PostedAt:  *now,
	}
	return message, nil
}

func (rm *RecordManager) updateMessage(m *types.Message, ctx context.Context) error {
	return rm.db.UpdateMessage(m, ctx)
}

func (rm *RecordManager) storeMessage(m *types.Message, ctx context.Context) error {
	return rm.db.StoreMessage(m, ctx)
}

func (rm *RecordManager) forwardMessage(m *types.Message, ctx context.Context) error {
	curi, err := rm.db.GetMsgChannelURI(m.SignetURI, ctx)
	if err != nil {
		return errors.New("aaaaaaaaaaaa " + err.Error())
	}
	return rm.broadcaster.BroadcastMessage(curi, m)
}

func (rm *RecordManager) validateMessage(mr *types.PostMessageRequest, ctx context.Context) (lmr *lex.MessageRecord, now *time.Time, handle *string, nonce []byte, err error) {
	lmr = &lex.MessageRecord{}
	if mr.SignetURI == nil {
		if mr.MessageID == nil || mr.ChannelURI == nil {
			err = errors.New("must provide a way to determine signet")
			return
		}
		signetUri, signetHandle, yorks := rm.db.QuerySignet(*mr.ChannelURI, *mr.MessageID, ctx)
		if yorks != nil {
			err = errors.New("i couldn't find the signet :c : " + yorks.Error())
			return
		}
		mr.SignetURI = &signetUri
		handle = &signetHandle
	} else {
		signetHandle, yorks := rm.db.QuerySignetHandle(*mr.SignetURI, ctx)
		if yorks != nil {
			err = errors.New("yorks skooby 💀" + yorks.Error())
			return
		}
		handle = &signetHandle
	}
	lmr.SignetURI = *mr.SignetURI
	lmr.Body = mr.Body
	if mr.Nick != nil {
		nick := *mr.Nick
		if atputils.ValidateLength(nick, 16) {
			err = errors.New("that nick is too long")
			return
		}
	}
	lmr.Nick = mr.Nick

	if mr.Color != nil {
		color := uint64(*mr.Color)
		if color > 16777215 {
			err = errors.New("that color is too big")
			return
		}
		lmr.Color = &color
	}

	nonce = mr.Nonce
	nowsyn := syntax.DatetimeNow()
	lmr.PostedAt = nowsyn.String()
	nt := nowsyn.Time()
	now = &nt
	return
}
