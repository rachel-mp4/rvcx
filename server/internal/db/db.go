package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"rvcx/internal/atputils"
	"rvcx/internal/types"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct {
	pool *pgxpool.Pool
}

func Init() (*Store, error) {
	pool, err := initialize()
	return &Store{pool}, err
}

func (s *Store) Close() {
	s.pool.Close()
}

func initialize() (*pgxpool.Pool, error) {
	dbuser := os.Getenv("POSTGRES_USER")
	dbpass := os.Getenv("POSTGRES_PASSWORD")
	dbhost := "localhost"
	dbport := os.Getenv("POSTGRES_PORT")
	dbdb := os.Getenv("POSTGRES_DB")
	dburl := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbuser, dbpass, dbhost, dbport, dbdb)
	pool, err := pgxpool.New(context.Background(), dburl)
	if err != nil {
		return nil, err
	}
	pingErr := pool.Ping(context.Background())
	if pingErr != nil {
		return nil, pingErr
	}
	fmt.Println("connected!")
	return pool, nil
}

func (s *Store) ResolveHandle(handle string, ctx context.Context) (string, error) {
	row := s.pool.QueryRow(ctx, `SELECT h.did FROM did_handles h WHERE h.handle = $1`, handle)
	var did string
	err := row.Scan(&did)
	if err != nil {
		return "", err
	}
	return did, nil
}

func (s *Store) ResolveDid(did string, ctx context.Context) (string, error) {
	row := s.pool.QueryRow(ctx, `SELECT h.handle FROM did_handles h WHERE h.did = $1`, did)
	var handle string
	err := row.Scan(&handle)
	if err != nil {
		return "", errors.New("error scanning row for handle: " + err.Error())
	}
	return handle, nil
}

func (s *Store) FullResolveDid(did string, ctx context.Context) (string, error) {
	hdl, err := s.ResolveDid(did, ctx)
	if err == nil {
		return hdl, nil
	}
	hdl, err = atputils.TryLookupDid(ctx, did)
	if err != nil {
		return "", errors.New("couldn't resolve: " + err.Error())
	}
	s.StoreDidHandle(did, hdl, ctx)
	return hdl, nil
}

func (s *Store) StoreDidHandle(did string, handle string, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO did_handles (
			handle,
			did
		) VALUES ($1, $2) ON CONFLICT (handle) DO NOTHING`, handle, did)
	if err != nil {
		return errors.New("error storing did/handle: " + err.Error())
	}
	return nil
}

func (s *Store) GetLastSeen(did string, ctx context.Context) (where *string, when *time.Time) {
	row := s.pool.QueryRow(ctx, `SELECT 
		s.channel_uri, m.posted_at 
		FROM messages m 
		JOIN signets s ON m.signet_uri = s.uri
		JOIN did_handles dh ON m.did = dh.did
		WHERE m.did = $1 AND dh.handle = s.author_handle
		ORDER BY m.posted_at DESC`, did)
	row.Scan(&where, &when)
	return
}

func (s *Store) GetMessages(channelURI string, limit int, cursor *int, ctx context.Context) ([]types.SignedMessageView, error) {
	queryFmt := `
		SELECT 
			m.uri, 
			m.did,
			dh.handle,
			p.display_name,
			p.status,
			p.color,
			p.avatar_cid,
			p.default_nick,
			m.body, 
			m.nick, 
			m.color, 
			s.uri,
			issuer_dh.handle,
			s.channel_uri,
			s.message_id,
			s.author_handle,
			s.started_at,
			m.posted_at
		FROM messages m 
		JOIN signets s ON m.signet_uri = s.uri
		JOIN did_handles dh ON m.did = dh.did
		LEFT JOIN profiles p ON m.did = p.did
		JOIN did_handles issuer_dh ON s.issuer_did = issuer_dh.did
		WHERE s.channel_uri = $2 AND dh.handle = s.author_handle %s
		ORDER BY s.message_id DESC
		LIMIT $1
		`
	var query string
	if cursor != nil {
		query = fmt.Sprintf(queryFmt, "AND s.message_id < $3")
		return s.evalGetMessages(query, ctx, limit, channelURI, *cursor)
	} else {
		query = fmt.Sprintf(queryFmt, "")
		return s.evalGetMessages(query, ctx, limit, channelURI)
	}
}

func (s *Store) evalGetMessages(query string, ctx context.Context, limit int, params ...any) ([]types.SignedMessageView, error) {
	args := []any{limit}
	args = append(args, params...)
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs = make([]types.SignedMessageView, 0)
	for rows.Next() {
		var msg types.SignedMessageView
		err := rows.Scan(
			&msg.URI,

			&msg.Author.DID,
			&msg.Author.Handle,
			&msg.Author.DisplayName,
			&msg.Author.Status,
			&msg.Author.Color,
			&msg.Author.Avatar,
			&msg.Author.DefaultNick,

			&msg.Body,
			&msg.Nick,
			&msg.Color,

			&msg.Signet.URI,
			&msg.Signet.IssuerHandle,
			&msg.Signet.ChannelURI,
			&msg.Signet.LrcId,
			&msg.Signet.AuthorHandle,
			&msg.Signet.StartedAt,

			&msg.PostedAt,
		)
		if err != nil {
			return nil, err
		}
		msgs = append(msgs, msg)
	}
	return msgs, nil
}

func (s *Store) GetChannelURI(handle string, title string, ctx context.Context) (string, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			channels.uri
		FROM channels
		LEFT JOIN did_handles ON channels.did = did_handles.did
		WHERE channels.title = $1 AND did_handles.handle = $2
		ORDER BY channels.created_at DESC
		LIMIT 1
		`, title, handle)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var uri string
	rows.Next()
	err = rows.Scan(&uri)
	if err != nil {
		return "", err
	}
	return uri, nil
}

type URIHost struct {
	URI    string
	Host   string
	Topic  string
	LastID uint32
}

func (s *Store) GetChannelURIs(ctx context.Context) ([]URIHost, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			channels.uri,
			channels.host,
			channels.topic
		FROM channels
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var urihosts = make([]URIHost, 0, 100)
	for rows.Next() {
		var urihost URIHost
		err := rows.Scan(&urihost.URI, &urihost.Host, &urihost.Topic)
		if err != nil {
			return nil, err
		}
		var maxMessageID uint32
		err = s.pool.QueryRow(ctx, `
			SELECT COALESCE(MAX(message_id), 0) 
			FROM signets 
			WHERE channel_uri = $1
			`, urihost.URI).Scan(&maxMessageID)
		if err != nil {
			return nil, err
		}
		urihost.LastID = maxMessageID
		urihosts = append(urihosts, urihost)
	}
	return urihosts, nil
}

func (s *Store) GetChannelViews(limit int, ctx context.Context) ([]types.ChannelView, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT 
			channels.uri,  
			channels.host, 
			channels.title, 
			channels.topic, 
			channels.created_at,
			did_handles.did,
			did_handles.handle,
			profiles.display_name,
			profiles.status,
			profiles.color,
			profiles.avatar_cid
		FROM channels
		LEFT JOIN profiles ON channels.did = profiles.did
		LEFT JOIN did_handles ON profiles.did = did_handles.did
		ORDER BY channels.created_at DESC
		LIMIT $1
		`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var chans = make([]types.ChannelView, 0, limit)
	for rows.Next() {
		var c types.ChannelView
		var p types.ProfileView
		err := rows.Scan(&c.URI, &c.Host, &c.Title, &c.Topic, &c.CreatedAt, &p.DID, &p.Handle, &p.DisplayName, &p.Status, &p.Color, &p.Avatar)
		if err != nil {
			return nil, err
		}
		c.Creator = p
		chans = append(chans, c)
	}
	return chans, nil
}

func (s *Store) GetChannelView(uri string, ctx context.Context) (*types.ChannelView, error) {
	row := s.pool.QueryRow(ctx, `
		SELECT 
			channels.uri,  
			channels.host, 
			channels.title, 
			channels.topic, 
			channels.created_at,
			did_handles.did,
			did_handles.handle,
			profiles.display_name,
			profiles.status,
			profiles.color,
			profiles.avatar_cid
		FROM channels
		LEFT JOIN profiles ON channels.did = profiles.did
		LEFT JOIN did_handles ON profiles.did = did_handles.did
		WHERE channels.uri = $1
		`, uri)
	var c types.ChannelView
	var p types.ProfileView
	err := row.Scan(&c.URI, &c.Host, &c.Title, &c.Topic, &c.CreatedAt, &p.DID, &p.Handle, &p.DisplayName, &p.Status, &p.Color, &p.Avatar)
	if err != nil {
		return nil, err
	}
	c.Creator = p
	return &c, nil
}
func (s *Store) GetChannelViewHR(handle string, rkey string, ctx context.Context) (*types.ChannelView, error) {
	did, err := s.ResolveHandle(handle, ctx)
	if err != nil {
		return nil, err
	}
	uri := fmt.Sprintf("at://%s/org.xcvr.feed.channel/%s", did, rkey)
	row := s.pool.QueryRow(ctx, `
		SELECT 
			channels.uri,  
			channels.host, 
			channels.title, 
			channels.topic, 
			channels.created_at,
			did_handles.did,
			did_handles.handle,
			profiles.display_name,
			profiles.status,
			profiles.color,
			profiles.avatar_cid
		FROM channels
		LEFT JOIN profiles ON channels.did = profiles.did
		LEFT JOIN did_handles ON profiles.did = did_handles.did
		WHERE channels.uri = $1
		`, uri)
	var c types.ChannelView
	var p types.ProfileView
	err = row.Scan(&c.URI, &c.Host, &c.Title, &c.Topic, &c.CreatedAt, &p.DID, &p.Handle, &p.DisplayName, &p.Status, &p.Color, &p.Avatar)
	if err != nil {
		return nil, err
	}
	c.Creator = p
	return &c, nil
}

func (s *Store) DeleteChannel(uri string, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `DELETE FROM channels WHERE uri = $1`, uri)
	return err
}
