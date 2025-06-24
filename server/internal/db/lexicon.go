package db

import (
	"strings"
	"context"
	"fmt"
	"xcvr-backend/internal/types"
	"errors"
)

func (s *Store) InitializeProfile(did string, handle string, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO profiles (
			did,
			display_name,
			default_nick,
			status,
			color
		) VALUES (
		$1, $2, $3, $4, $5
		) ON CONFLICT (did) DO NOTHING
		`, did, handle, "wanderer", "just setting up my xcvr", 12517472)
	if err != nil {
		return errors.New("i'm not sure what happened: " + err.Error())
	}
	return nil
}

type ProfileUpdate struct {
	DID          string
	Name         *string
	UpdateName   bool
	Nick         *string
	UpdateNick   bool
	Status       *string
	UpdateStatus bool
	Avatar       *string
	UpdateAvatar bool
	Mime         *string
	UpdateMime   bool
	Color        *uint32
	UpdateColor  bool
}

func (s *Store) UpdateProfile(to ProfileUpdate, ctx context.Context) error {
	setParts := []string{}
	args := []any{to.DID}
	idx := 2
	if to.UpdateName {
		setParts = append(setParts, fmt.Sprintf("display_name = $%d", idx))
		args = append(args, to.Name)
		idx += 1
	}
	if to.UpdateNick {
		setParts = append(setParts, fmt.Sprintf("default_nick = $%d", idx))
		args = append(args, to.Nick)
		idx += 1
	}
	if to.UpdateStatus {
		setParts = append(setParts, fmt.Sprintf("status = $%d", idx))
		args = append(args, to.Status)
		idx += 1
	}
	if to.UpdateAvatar {
		setParts = append(setParts, fmt.Sprintf("avatar_cid = $%d", idx))
		args = append(args, to.Avatar)
		idx += 1
	}
	if to.UpdateMime {
		setParts = append(setParts, fmt.Sprintf("avatar_mime = $%d", idx))
		args = append(args, to.Mime)
		idx += 1
	}
	if to.UpdateColor {
		setParts = append(setParts, fmt.Sprintf("color = $%d", idx))
		args = append(args, to.Color)
		idx += 1
	}
	if idx == 2 {
		return nil
	}
	sql := fmt.Sprintf("UPDATE profiles SET %s WHERE did = $1", 
		strings.Join(setParts, ", "))
	_, err := s.pool.Exec(ctx, sql, args...)
	if err != nil {
		return errors.New("error updating profile: " + err.Error())
	}
	return nil
}

func (s *Store) GetProfileView(did string, ctx context.Context) (*types.ProfileView, error) {
	row := s.pool.QueryRow(ctx, `SELECT 
		p.display_name,
		p.default_nick,
		p.status,
		p.avatar_cid,
		p.color
		FROM profiles p
		WHERE p.did = $1
		`, did)
	var p types.ProfileView
	p.DID = did
	err := row.Scan(&p.DisplayName, 
		&p.DefaultNick, 
		&p.Status, 
		&p.Avatar, 
		&p.Color)
	if err != nil {
		return nil, errors.New("error scanning profile: " + err.Error())
	}
	return &p, nil
}





