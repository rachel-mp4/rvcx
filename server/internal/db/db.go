package db

import (
	"context"
	"fmt"
	"xcvr-backend/internal/types"
	"os"

	"github.com/jackc/pgx/v5"
)

func Init() (*pgx.Conn, error) {
	dbuser := os.Getenv("POSTGRES_USER")
	dbpass := os.Getenv("POSTGRES_PASSWORD")
	dbhost := "localhost"
	dbport := os.Getenv("POSTGRES_PORT")
	dbdb := os.Getenv("POSTGRES_DB")
	dburl := fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", dbuser, dbpass, dbhost, dbport, dbdb)
	conn, err := pgx.Connect(context.Background(), dburl)
	if err != nil {
		return conn, err
	}
	pingErr := conn.Ping(context.Background())
	if pingErr != nil {
		return conn, pingErr
	}
	fmt.Println("connected!")
	return conn, nil
}

func GetMessages(channelURI string, limit int,ctx context.Context, db *pgx.Conn) ([]types.Message, error) {
	rows, err := db.Query(ctx, `
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

func GetChannels(limit int, ctx context.Context, db *pgx.Conn) ([]types.Channel, error) {
	rows, err := db.Query(ctx, `
		SELECT 
			c.uri, c.did, c.host, c.title, c.topic, c.created_at
		FROM channels c
		ORDER BY s.message_id DESC
		LIMIT $2
		`, limit)
	if err != nil {
		return nil, err
	}
	var chans = make([]types.Channel, 0, limit) 
	for rows.Next() {
		var c types.Channel
		err := rows.Scan(&c.URI, &c.DID, &c.Host, &c.Title, &c.Topic, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		chans = append(chans, c)
	}
	return chans, nil
}

func GetChannelViews(limit int, ctx context.Context, db *pgx.Conn) ([]types.ChannelView, error) {
	rows, err := db.Query(ctx, `
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

