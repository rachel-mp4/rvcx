package recordmanager

import (
	"context"
	"errors"
	"fmt"
	atoauth "github.com/bluesky-social/indigo/atproto/auth/oauth"
	"github.com/bluesky-social/indigo/atproto/syntax"
	lexutil "github.com/bluesky-social/indigo/lex/util"
	"mime/multipart"
	"os"
	"rvcx/internal/atputils"
	"rvcx/internal/lex"
	"rvcx/internal/oauth"
	"rvcx/internal/types"
	"time"
)

func (rm *RecordManager) PostImage(cs *atoauth.ClientSession, file multipart.File, fileHeader *multipart.FileHeader, ctx context.Context) (*lexutil.BlobSchema, error) {
	return oauth.UploadBLOB(cs, file, fileHeader, ctx)
}

func (rm *RecordManager) AddImageToCache(did string, cid string, ctx context.Context) (string, error) {
	ib, err := rm.db.IsBanned(did, ctx)
	if err != nil {
		return "", err
	}
	if ib {
		return "", errors.New("user banned")
	}
	uploadDir := "./uploads"
	_, err = os.Stat(uploadDir)
	if os.IsNotExist(err) {
		os.Mkdir(uploadDir, 0755)
	}

	imgPath := fmt.Sprintf("%s/%s%s", uploadDir, did, cid)
	_, err = os.Stat(imgPath)
	if err != nil {
		blob, err := atputils.SyncGetBlob(did, cid, ctx)
		if err != nil {
			return "", err
		}
		file, err := os.Create(imgPath)
		if err != nil {
			return "", err
		}
		_, err = file.Write(blob)
		if err != nil {
			return "", err
		}
	}
	return imgPath, nil
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
	err = rm.forwardImage(img, ctx)
	if err != nil {
		return errors.New("YIEKRSa, " + err.Error())
	}
	return nil
}

func (rm *RecordManager) forwardImage(i *types.Image, ctx context.Context) error {
	curi, err := rm.db.GetMsgChannelURI(i.SignetURI, ctx)
	if err != nil {
		return errors.New("failed to get curi: " + err.Error())
	}
	return rm.broadcaster.BroadcastImage(curi, i)
}

func (rm *RecordManager) validateImageRecord(mr *types.ParseMediaRequest, ctx context.Context) (*lex.MediaRecord, *time.Time, error) {
	var imr lex.MediaRecord
	if mr.SignetURI == nil {
		if mr.ChannelURI == nil || mr.MessageID == nil {
			return nil, nil, errors.New("not enough info!")
		}
		suri, _, err := rm.db.QuerySignet(*mr.ChannelURI, *mr.MessageID, ctx)
		if err != nil {
			return nil, nil, errors.New("failed to get signet!")
		}
		mr.SignetURI = &suri
	}
	imr.SignetURI = *mr.SignetURI
	imr.Nick = mr.Nick
	cptr := mr.Color
	if cptr != nil {
		cnum := uint64(*cptr)
		imr.Color = &cnum
	}
	imr.Image = mr.Image
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
	if imr.Image != nil {
		img.Alt = imr.Image.Alt
		if imr.Image.Blob != nil {
			img.BlobMIME = &imr.Image.Blob.MimeType
			icid := imr.Image.Blob.Ref.String()
			img.BlobCID = &icid
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
