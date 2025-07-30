package recordmanager

import (
	"context"
	"errors"
	"rvcx/internal/atputils"
	"rvcx/internal/db"
	"rvcx/internal/lex"
	"rvcx/internal/types"
)

func (rm *RecordManager) CreateInitialProfile(did string, id int, ctx context.Context) error {
	nick := "wanderer"
	status := "just setting up my xcvr"
	color := uint64(3602605)
	handle, err := rm.db.ResolveDid(did, ctx)
	if err != nil {
		return errors.New("i couldn't find the handle, so i couldn't create default profile record. gootbye")
	}

	p, err := rm.createProfile(&handle, &nick, &status, &color, id, ctx)
	if err != nil {
		return errors.New("AAAAA error creating profile")
	}
	rm.log.Deprintln("initializing profile....")
	err = rm.db.InitializeProfile(did, p.DisplayName, p.DefaultNick, p.Status, p.Color, ctx)
	if err != nil {
		return errors.New("failed to initialize profile: " + err.Error())
	}
	return nil

}

func (rm *RecordManager) PostProfile(did string, id int, ctx context.Context, p *types.PostProfileRequest) error {
	pu, err := rm.validateProfile(did, p)
	if err != nil {
		return errors.New("couldn't validate profile: " + err.Error())
	}
	err = rm.updateProfile(p.DisplayName, p.DefaultNick, p.Status, p.Color, id, ctx)
	if err != nil {
		return errors.New("couldn't create profile: " + err.Error())
	}
	err = rm.storeProfile(pu, ctx)
	if err != nil {
		return errors.New("couldn't store profile: " + err.Error())
	}
	return nil
}

func (rm *RecordManager) storeProfile(pu *db.ProfileUpdate, ctx context.Context) error {
	err := rm.db.UpdateProfile(pu, ctx)
	if err != nil {
		return errors.New("error updating profile: " + err.Error())
	}
	return nil
}

func (rm *RecordManager) updateProfile(name *string, nick *string, status *string, color *uint64, id int, ctx context.Context) error {
	profilerecord := &lex.ProfileRecord{
		DisplayName: name,
		DefaultNick: nick,
		Status:      status,
		Color:       color,
	}
	client, err := rm.getClient(id, ctx)
	if err != nil {
		return err
	}
	_, err = client.UpdateXCVRProfile(profilerecord, ctx)
	if err != nil {
		return err
	}
	return nil
}

func (rm *RecordManager) createProfile(name *string, nick *string, status *string, color *uint64, id int, ctx context.Context) (*lex.ProfileRecord, error) {
	profilerecord := &lex.ProfileRecord{
		DisplayName: name,
		DefaultNick: nick,
		Status:      status,
		Color:       color,
	}
	client, err := rm.getClient(id, ctx)
	if err != nil {
		return nil, err
	}
	p, err := client.CreateXCVRProfile(profilerecord, ctx)
	if err != nil {
		return nil, errors.New("failed to create profile: " + err.Error())
	}
	return p, nil
}

func (rm *RecordManager) validateProfile(did string, p *types.PostProfileRequest) (*db.ProfileUpdate, error) {
	var pu db.ProfileUpdate
	pu.DID = did
	if p.DisplayName != nil {
		if atputils.ValidateGraphemesAndLength(*p.DisplayName, 64, 640) {
			return nil, errors.New("displayname too long")
		}
		pu.Name = p.DisplayName
		pu.UpdateName = true
	}
	if p.DefaultNick != nil {
		if atputils.ValidateLength(*p.DefaultNick, 16) {
			return nil, errors.New("nick too long")
		}
		pu.Nick = p.DefaultNick
		pu.UpdateNick = true
	}
	if p.Status != nil {
		if atputils.ValidateGraphemesAndLength(*p.Status, 640, 6400) {
			return nil, errors.New("status too long")
		}
		pu.Status = p.Status
		pu.UpdateStatus = true
	}
	if p.Avatar != nil {
		// TODO think about how to do avatars!
		pu.Avatar = p.Avatar
		pu.UpdateAvatar = true
	}
	if p.Color != nil {
		if *p.Color > 16777215 || *p.Color < 0 {
			return nil, errors.New("color out of bounds")
		}
		pu.Color = p.Color
		pu.UpdateColor = true
	}
	return &pu, nil
}
