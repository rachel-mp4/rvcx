package db

import (
	"context"
	"errors"
	"fmt"
	"rvcx/internal/types"
	"strings"
)

func (s *Store) InitializeProfile(did string,
	displayname *string,
	defaultnick *string,
	status *string,
	color *uint64,
	ctx context.Context) error {
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
		`, did, displayname, defaultnick, status, color)
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
	Color        *uint64
	UpdateColor  bool
}

func (s *Store) UpdateProfile(to *ProfileUpdate, ctx context.Context) error {
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

func (s *Store) DeleteProfile(did string, cid string, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM profiles p WHERE p.DID = $1 AND p.CID = $2
		`, did, cid)
	return err
}

func (s *Store) GetProfileView(did string, ctx context.Context) (*types.ProfileView, error) {
	row := s.pool.QueryRow(ctx, `SELECT 
		dh.handle,
		p.display_name,
		p.status,
		p.color,
		p.avatar_cid,
		p.default_nick
		FROM profiles p
		JOIN did_handles dh ON p.did = dh.did
		WHERE p.did = $1
		`, did)
	var p types.ProfileView
	p.DID = did
	err := row.Scan(
		&p.Handle,
		&p.DisplayName,
		&p.Status,
		&p.Color,
		&p.Avatar,
		&p.DefaultNick)
	if err != nil {
		return nil, errors.New("error scanning profile: " + err.Error())
	}
	return &p, nil
}

func (s *Store) StoreChannel(channel *types.Channel, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO channels (
		  uri,
			cid,
			did,
			host,
			title,
			topic,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		) ON CONFLICT (uri) DO NOTHING
		`, channel.URI, channel.CID, channel.DID, channel.Host, channel.Title, channel.Topic, channel.CreatedAt)
	return err
}

func (s *Store) UpdateChannel(channel *types.Channel, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO channels (
		  uri,
			cid,
			did,
			host,
			title,
			topic,
			created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7
		)`, channel.URI, channel.CID, channel.DID, channel.Host, channel.Title, channel.Topic, channel.CreatedAt)
	return err
}

func (s *Store) DeleteMessage(uri string, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM messages m WHERE m.uri = $1
		`, uri)
	return err
}

func (s *Store) StoreMessage(message *types.Message, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO messages (
		  uri,
			cid,
			did,
			signet_uri,
			body,
			nick,
			color,
			posted_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) ON CONFLICT (uri) DO NOTHING
		`, message.URI, message.CID, message.DID, message.SignetURI, message.Body, message.Nick, message.Color, message.PostedAt)
	return err
}

func (s *Store) UpdateMessage(message *types.Message, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO messages (
		  uri,
			cid,
			did,
			signet_uri,
			body,
			nick,
			color,
			posted_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8
		) ON CONFLICT (uri) DO NOTHING
		`, message.URI, message.CID, message.DID, message.SignetURI, message.Body, message.Nick, message.Color, message.PostedAt)
	return err
}

func (s *Store) QuerySignet(channelUri string, id uint32, ctx context.Context) (signetUri string, signetHandle string, err error) {
	row := s.pool.QueryRow(ctx, `SELECT s.uri, s.author_handle FROM signets s WHERE s.channel_uri = $1 AND s.message_id = $2`, channelUri, id)
	err = row.Scan(&signetUri, &signetHandle)
	if err != nil {
		err = errors.New("error scanning: " + err.Error())
	}
	return
}

func (s *Store) QuerySignetHandle(uri string, ctx context.Context) (string, error) {
	row := s.pool.QueryRow(ctx, `SELECT s.author_handle FROM signets s WHERE s.uri = $1`, uri)
	var handle string
	err := row.Scan(&handle)
	if err != nil {
		return "", errors.New("BOBOBOBOBOBOL " + err.Error())
	}
	return handle, nil
}

func (s *Store) QuerySignetChannelIdNum(uri string, ctx context.Context) (channelUri string, messageID uint32, err error) {
	row := s.pool.QueryRow(ctx, `SELECT s.channel_uri, s.message_id FROM signets s WHERE s.uri = $1`, uri)
	err = row.Scan(&channelUri, &messageID)
	if err != nil {
		err = errors.New("BOBOBOBOBOBOL " + err.Error())
	}
	return
}

func (s *Store) GetMsgChannelURI(signetURI string, ctx context.Context) (string, error) {
	row := s.pool.QueryRow(ctx, `SELECT s.channel_uri FROM signets s WHERE s.uri = $1`, signetURI)
	var channelURI string
	err := row.Scan(&channelURI)
	if err != nil {
		return "", errors.New("error scanning: " + err.Error())
	}
	return channelURI, nil
}

func (s *Store) StoreSignet(signet *types.Signet, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO signets (
			uri,
			issuer_did,
			author_handle,
			channel_uri,
			message_id,
			cid,
			started_at
		) VALUES (
		$1, $2, $3, $4, $5, $6, $7
		) ON CONFLICT (uri) DO NOTHING
		`, signet.URI, signet.IssuerDID, signet.AuthorHandle, signet.ChannelURI, signet.MessageID, signet.CID, signet.StartedAt)
	if err != nil {
		err = errors.New("SOMETHING BAD HAPPENED: " + err.Error())
	}
	return err
}

func (s *Store) UpdateSignet(signet *types.Signet, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		INSERT INTO signets (
			uri,
			issuer_did,
			AuthorHandle,
			channel_uri,
			message_id,
			cid,
			started_at
		) VALUES (
		$1, $2, $3, $4, $5, $6, $7
		)
		`, signet.URI, signet.IssuerDID, signet.AuthorHandle, signet.ChannelURI, signet.MessageID, signet.CID, signet.StartedAt)
	if err != nil {
		err = errors.New("SOMETHING BAD HAPPENED: " + err.Error())
	}
	return err
}

func (s *Store) DeleteSignet(uri string, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `
		DELETE FROM signets s WHERE s.uri = $1
		`, uri)
	return err
}
