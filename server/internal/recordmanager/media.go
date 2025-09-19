package recordmanager

import (
	"context"
	"errors"
	atoauth "github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"mime/multipart"
	"rvcx/internal/lex"
	"rvcx/internal/oauth"
	"rvcx/internal/types"
	"time"
)

func (rm *RecordManager) PostImage(cs *atoauth.ClientSession, file multipart.File, ctx context.Context) (*lexutil.BlobSchema, error) {
	return oauth.UploadBLOB(cs, file, ctx)
}

func (rm *RecordManager) PostMedia(cs *atoauth.ClientSession, mr *types.ParseMediaRequest, ctx context.Context) error {
	switch mr.Type {
	case "image":
		return rm.postImageRecord(cs, mr, ctx)
	default:
		return nil
	}
}

func (rm *RecordManager) postImageRecord(cs *atoauth.ClientSession, mr *types.ParseMediaRequest, ctx context.Context) error {
	imr, now, err := rm.validateImageRecord(mr)
	if err != nil {
		return errors.New("coudlnt validate media record: " + err.Error())
	}
	img, err := rm.createImageRecord(cs, imr, now, ctx)
	if err != nil {
		return errors.New("coudlnt validate media record: " + err.Error())
	}
	err = rm.db.StoreImage(img, ctx)
	if err != nil {
		return errors.New("beeped that up!: " + err.Error())
	}
	return nil
}

func (rm *RecordManager) validateImageRecord(mr *types.ParseMediaRequest) (*lex.MediaRecord, *time.Time, error) {
	var imr lex.MediaRecord
	imr.SignetURI = mr.SignetURI
	imr.Nick = mr.Nick
	cptr := mr.Color
	if cptr != nil {
		cnum := uint64(*cptr)
		imr.Color = &cnum
	}
	imr.Media.Image = mr.Image
	nowsyn := syntax.DatetimeNow()
	imr.PostedAt = nowsyn.String()
	nt := nowsyn.Time()
	now := &nt
	return &imr, now, nil
}

func (rm *RecordManager) createImageRecord(cs *atoauth.ClientSession, imr *lex.MediaRecord, now *time.Time, ctx context.Context) (*types.Image, error) {
	uri, cid, err := oauth.CreateXCVRMedia(cs, imr, ctx)
	if err != nil {
		return nil, errors.New("beeped up: " + err.Error())
	}
	var img types.Image
	img.URI = uri
	img.DID = cs.Data.AccountDID.String()
	img.SignetURI = imr.SignetURI
	if imr.Media.Image != nil {
		img.Alt = imr.Media.Image.Alt
		if imr.Media.Image.Image != nil {
			img.ImageMIME = &imr.Media.Image.Image.MimeType
			icid := imr.Media.Image.Image.Ref.String()
			img.ImageCID = &icid
		}
	}
	img.Nick = imr.Nick
	img.CID = cid
	if imr.Color != nil {
		c := uint32(*imr.Color)
		img.Color = &c
	}
	if now != nil {
		img.PostedAt = *now
	} else {
		img.PostedAt = time.Now()
	}
	return &img, nil
}
