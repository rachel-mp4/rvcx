package atplistener

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/bluesky-social/indigo/atproto/syntax"
	"github.com/bluesky-social/jetstream/pkg/client"
	"github.com/bluesky-social/jetstream/pkg/client/schedulers/sequential"
	"github.com/bluesky-social/jetstream/pkg/models"
	"time"
	"xcvr-backend/internal/db"
	"xcvr-backend/internal/lex"
	"xcvr-backend/internal/log"
	"xcvr-backend/internal/types"
)

type Consumer struct {
	cfg     *client.ClientConfig
	logger  *log.Logger
	handler *handler
}

type handler struct {
	db *db.Store
}

func NewConsumer(jsAddr string, l *log.Logger, db *db.Store) *Consumer {
	cfg := client.DefaultClientConfig()
	if jsAddr != "" {
		cfg.WebsocketURL = jsAddr
	}
	cfg.WantedCollections = []string{
		"org.xcvr.actor.profile",
		"org.xcvr.feed.channel",
		"org.xcvr.lrc.message",
		"org.xcvr.lrc.signet",
	}
	cfg.WantedDids = []string{}
	return &Consumer{
		cfg:     cfg,
		logger:  l,
		handler: &handler{db: db},
	}
}

func (c *Consumer) Consume(ctx context.Context) error {
	scheduler := sequential.NewScheduler("jetstream_localdev", c.logger.Slog, c.handler.HandleEvent)
	defer scheduler.Shutdown()
	client, err := client.NewClient(c.cfg, c.logger.Slog, scheduler)
	if err != nil {
		return errors.New("failed to create client: " + err.Error())
	}
	cursor := time.Now().Add(1 * -time.Minute).UnixMicro()
	err = client.ConnectAndRead(ctx, &cursor)
	if err != nil {
		return errors.New("error connecting and reading: " + err.Error())
	}
	return nil
}

func (h *handler) HandleEvent(ctx context.Context, event *models.Event) error {
	if event.Commit == nil {
		return nil
	}

	switch event.Commit.Collection {
	case "org.xcvr.actor.profile":
		return h.handleProfile(ctx, event)
	case "org.xcvr.feed.channel":
		return h.handleChannel(ctx, event)
	}
	return nil
}

func (h *handler) handleProfile(ctx context.Context, event *models.Event) error {
	var pr lex.ProfileRecord
	err := json.Unmarshal(event.Commit.Record, &pr)
	if err != nil {
		return errors.New("error unmarshaling: " + err.Error())
	}
	to := db.ProfileUpdate{
		DID: event.Did,
	}
	to.UpdateName = pr.DisplayName != nil
	to.Name = pr.DisplayName
	to.UpdateNick = pr.DefaultNick != nil
	to.Nick = pr.DefaultNick
	to.UpdateStatus = pr.Status != nil
	to.Status = pr.Status
	to.UpdateColor = pr.Color != nil
	to.Color = pr.Color
	return h.db.UpdateProfile(to, ctx)
}

func (h *handler) handleChannel(ctx context.Context, event *models.Event) error {
	var cr lex.ChannelRecord
	err := json.Unmarshal(event.Commit.Record, &cr)
	if err != nil {
		return errors.New("error unmarshl: " + err.Error())
	}
	then, err := syntax.ParseDatetimeTime(cr.CreatedAt)
	if err != nil {
		then = time.Now()
	}
	channel := types.Channel{
		URI:       fmt.Sprintf("at://%s/org.xcvr.feed.channel/%s", event.Did, event.Commit.RKey),
		CID:       event.Commit.CID,
		DID:       event.Did,
		Host:      cr.Host,
		Title:     cr.Title,
		Topic:     cr.Topic,
		CreatedAt: then,
	}
	return h.db.StoreChannel(channel, ctx)
}
