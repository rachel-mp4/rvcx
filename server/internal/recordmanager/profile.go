package recordmanager

import (
	"context"
	"errors"
	"rvcx/internal/atputils"
	"rvcx/internal/lex"
	"rvcx/internal/oauth"
	"rvcx/internal/types"

	atoauth "github.com/bluesky-social/indigo/atproto/auth/oauth"
)

func (rm *RecordManager) AcceptProfile(p lex.ProfileRecord, did string, ctx context.Context) error {
	err := rm.storeProfile(did, &p, ctx)
	if err != nil {
		return errors.New("failed to store profile: " + err.Error())
	}
	return nil
}

func (rm *RecordManager) DeleteProfile(did string, cid string, ctx context.Context) error {
	return rm.db.DeleteProfile(did, cid, ctx)
}

func (rm *RecordManager) CreateInitialProfile(sessData *atoauth.ClientSessionData, ctx context.Context) error {
	nick := "wanderer"
	status := "just setting up my xcvr"
	color := uint64(3702605)
	handle, err := rm.db.FullResolveDid(sessData.AccountDID.String(), ctx)
	if err != nil {
		return errors.New("i couldn't find the handle, so i couldn't create default profile record. gootbye")
	}

	p, err := rm.createProfile(&handle, &nick, &status, &color, sessData, ctx)
	if err != nil {
		return errors.New("AAAAA error creating profile" + err.Error())
	}
	rm.log.Deprintln("initializing profile....")
	err = rm.db.InitializeProfile(sessData.AccountDID.String(), p.DisplayName, p.DefaultNick, p.Status, p.Color, ctx)
	if err != nil {
		return errors.New("failed to initialize profile: " + err.Error())
	}
	return nil

}

func (rm *RecordManager) PostProfile(cs *atoauth.ClientSession, ctx context.Context, p *types.PostProfileRequest) error {
	err := rm.validateProfile(p)
	if err != nil {
		return errors.New("couldn't validate profile: " + err.Error())
	}
	pr, err := rm.updateProfile(cs, p.DisplayName, p.DefaultNick, p.Status, p.Color, ctx)
	if err != nil {
		return errors.New("couldn't create profile: " + err.Error())
	}
	err = rm.storeProfile(cs.Data.AccountDID.String(), pr, ctx)
	if err != nil {
		return errors.New("couldn't store profile: " + err.Error())
	}
	return nil
}

func (rm *RecordManager) storeProfile(did string, p *lex.ProfileRecord, ctx context.Context) error {
	err := rm.db.UpdateProfile(did, p.DisplayName, p.DefaultNick, p.Status, p.Color, ctx)
	if err != nil {
		return errors.New("error updating profile: " + err.Error())
	}
	return nil
}

func (rm *RecordManager) updateProfile(cs *atoauth.ClientSession, name *string, nick *string, status *string, color *uint64, ctx context.Context) (*lex.ProfileRecord, error) {
	profilerecord := &lex.ProfileRecord{
		DisplayName: name,
		DefaultNick: nick,
		Status:      status,
		Color:       color,
	}
	pr, err := oauth.UpdateXCVRProfile(cs, profilerecord, ctx)
	if err != nil {
		return nil, err
	}
	return pr, nil
}

func (rm *RecordManager) createProfile(name *string, nick *string, status *string, color *uint64, sessData *atoauth.ClientSessionData, ctx context.Context) (*lex.ProfileRecord, error) {
	profilerecord := &lex.ProfileRecord{
		DisplayName: name,
		DefaultNick: nick,
		Status:      status,
		Color:       color,
	}
	client, err := rm.service.ResumeSession(ctx, sessData.AccountDID, sessData.SessionID)
	if err != nil {
		return nil, err
	}
	p, err := oauth.CreateXCVRProfile(client, profilerecord, ctx)
	if err != nil {
		return nil, errors.New("failed to create profile: " + err.Error())
	}
	return p, nil
}

func (rm *RecordManager) validateProfile(p *types.PostProfileRequest) error {
	if p.DisplayName != nil {
		if atputils.ValidateGraphemesAndLength(*p.DisplayName, 64, 640) {
			return errors.New("displayname too long")
		}
	}
	if p.DefaultNick != nil {
		if atputils.ValidateLength(*p.DefaultNick, 16) {
			return errors.New("nick too long")
		}
	}
	if p.Status != nil {
		if atputils.ValidateGraphemesAndLength(*p.Status, 640, 6400) {
			return errors.New("status too long")
		}
	}
	if p.Avatar != nil {
		// TODO think about how to do avatars!
	}
	if p.Color != nil {
		if *p.Color > 16777215 || *p.Color < 0 {
			return errors.New("color out of bounds")
		}
	}
	return nil
}
