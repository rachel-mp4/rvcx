package db

import (
	"context"
	"errors"
	"fmt"
	"os"
	"xcvr-backend/internal/types"

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
	rows, err := s.pool.Query(ctx, `
		SELECT
			h.did
		FROM did_handles h
		WHERE h.handle = $1
		LIMIT 1
	`, handle)
	if err != nil {
		return "", err
	}
	defer rows.Close()
	var did string
	for rows.Next() {
		err := rows.Scan(&did)
		if err != nil {
			return "", err
		}
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

func (s *Store) StoreDidHandle(did string, handle string, ctx context.Context) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO did_handles (
			handle
			did
		) VALUES ($1, $2)`, handle, did)
	if err != nil {
		return errors.New("error storing did/handle: " + err.Error())
	}
	return nil
}

func (s *Store) GetMessages(channelURI string, limit int, ctx context.Context) ([]types.Message, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT 
			m.uri, m.did, m.signet_uri, m.body, m.nick, m.color, m.posted_at
		FROM messages m 
		JOIN signets s ON m.signet_uri = s.uri
		WHERE s.channel_uri = $1
		ORDER BY s.message_id DESC
		LIMIT $2
		`, channelURI, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var msgs = make([]types.Message, 0, limit)
	for rows.Next() {
		var msg types.Message
		err := rows.Scan(&msg.URI, &msg.DID, &msg.SignetURI, &msg.Body, &msg.Nick, &msg.PostedAt)
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
	URI  string
	Host string
}

func (s *Store) GetChannelURIs(ctx context.Context) ([]URIHost, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT
			channels.uri,
			channels.host
		FROM channels
		`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var urihosts = make([]URIHost, 0, 100)
	for rows.Next() {
		var urihost URIHost
		err := rows.Scan(&urihost.URI, &urihost.Host)
		if err != nil {
			return nil, err
		}
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
