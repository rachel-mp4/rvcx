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
	"rvcx/internal/atputils"
	"rvcx/internal/db"
	"rvcx/internal/lex"
	"rvcx/internal/log"
	"rvcx/internal/oauth"
	"rvcx/internal/recordmanager"
	"rvcx/internal/types"
	"time"
)

type Consumer struct {
	cfg     *client.ClientConfig
	logger  *log.Logger
	handler *handler
}

type handler struct {
	db  *db.Store
	rm  *recordmanager.RecordManager
	l   *log.Logger
	cli *oauth.PasswordClient
}

func NewConsumer(jsAddr string, l *log.Logger, db *db.Store, cli *oauth.PasswordClient, rm *recordmanager.RecordManager) *Consumer {
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
		handler: &handler{db: db, l: l, cli: cli, rm: rm},
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
	err := h.ensureIKnowYou(event.Did, ctx)
	if err != nil {
		return err
	}

	switch event.Commit.Collection {
	case "org.xcvr.actor.profile":
		return h.handleProfile(ctx, event)
	case "org.xcvr.feed.channel":
		return h.handleChannel(ctx, event)
	case "org.xcvr.lrc.message":
		return h.handleMessage(ctx, event)
	case "org.xcvr.lrc.signet":
		return h.handleSignet(ctx, event)
	}
	return nil
}

func (h *handler) handleProfile(ctx context.Context, event *models.Event) error {
	h.l.Deprintln("handling profile")
	switch event.Commit.Operation {
	case "create", "update":
		return h.handleProfileCreateUpdate(ctx, event)
	case "delete":
		return h.handleProfileDelete(ctx, event)
	}
	return errors.New("unsupported commit operation")
}

func (h *handler) handleProfileCreateUpdate(ctx context.Context, event *models.Event) error {
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
	return h.db.UpdateProfile(&to, ctx)
}

func (h *handler) handleProfileDelete(ctx context.Context, event *models.Event) error {
	return h.db.DeleteProfile(event.Did, event.Commit.CID, ctx)
}

func (h *handler) handleChannel(ctx context.Context, event *models.Event) error {
	h.l.Deprintln("handling channel")
	switch event.Commit.Operation {
	case "create":
		return h.handleChannelCreate(ctx, event)
	case "update":
		return h.handleChannelUpdate(ctx, event)
	case "delete":
		return h.handleChannelDelete(ctx, event)
	}
	return nil
}

func (h *handler) handleChannelCreate(ctx context.Context, event *models.Event) error {
	channel, err := parseChannelRecord(event)
	if err != nil {
		return errors.New("i couldn't create the channel: " + err.Error())
	}
	return h.rm.AcceptChannel(channel, ctx)
}

func (h *handler) handleChannelUpdate(ctx context.Context, event *models.Event) error {
	channel, err := parseChannelRecord(event)
	if err != nil {
		return errors.New("i couldn't create the channel: " + err.Error())
	}
	return h.db.UpdateChannel(channel, ctx)
}

func parseChannelRecord(event *models.Event) (*types.Channel, error) {
	var cr lex.ChannelRecord
	err := json.Unmarshal(event.Commit.Record, &cr)
	if err != nil {
		return nil, errors.New("error unmarshl: " + err.Error())
	}
	then, err := syntax.ParseDatetimeTime(cr.CreatedAt)
	if err != nil {
		then = time.Now()
	}
	channel := types.Channel{
		URI:       URI(event),
		CID:       event.Commit.CID,
		DID:       event.Did,
		Host:      cr.Host,
		Title:     cr.Title,
		Topic:     cr.Topic,
		CreatedAt: then,
	}
	return &channel, nil
}

func (h *handler) handleChannelDelete(ctx context.Context, event *models.Event) error {
	return h.db.DeleteChannel(URI(event), ctx)
}

func (h *handler) handleMessage(ctx context.Context, event *models.Event) error {
	h.l.Deprintln("handling message")
	switch event.Commit.Operation {
	case "create":
		return h.handleMessageCreate(ctx, event)
	case "update":
		return h.handleMessageUpdate(ctx, event)
	case "delete":
		return h.handleMessageDelete(ctx, event)
	}
	return errors.New("unimplemented Operation")
}

func (h *handler) handleMessageCreate(ctx context.Context, event *models.Event) error {
	message, err := parseMessageRecord(event)
	if err != nil {
		return errors.New("error parsing: " + err.Error())
	}
	return h.db.StoreMessage(message, ctx)
}

func (h *handler) handleMessageUpdate(ctx context.Context, event *models.Event) error {
	message, err := parseMessageRecord(event)
	if err != nil {
		return errors.New("error parsing: " + err.Error())
	}
	host, _ := atputils.DidFromUri(message.SignetURI)
	rkey, err := atputils.RkeyFromUri(message.SignetURI)
	if err != nil {
		return errors.New("i think the record is borked ngl")
	}
	if host == atputils.GetMyDid() {
		dne, err := h.cli.DeleteXCVRSignet(rkey, ctx)
		if err != nil {
			if dne {
				err = h.db.DeleteSignet(message.SignetURI, ctx)
				if err != nil {
					return errors.New("a lot of stuff happened yikers!" + err.Error())
				}
				return nil
			}
			return errors.New("failed to delete signet after infetterance: " + err.Error())
		}
		err = h.db.DeleteSignet(message.SignetURI, ctx)
		if err != nil {
			return errors.New("i deleted the signet, however i couldn't delete it from my db: " + err.Error())
		}
		return nil
	}
	return h.db.UpdateMessage(message, ctx)
}

func (h *handler) handleMessageDelete(ctx context.Context, event *models.Event) error {
	return h.db.DeleteMessage(URI(event), ctx)
}

func parseMessageRecord(event *models.Event) (*types.Message, error) {
	var mr lex.MessageRecord
	err := json.Unmarshal(event.Commit.Record, &mr)
	if err != nil {
		return nil, errors.New("error unmarshl: " + err.Error())
	}
	then, err := syntax.ParseDatetimeTime(mr.PostedAt)
	if err != nil {
		then = time.Now()
	}
	var color *uint32
	if mr.Color != nil {
		c := uint32(*mr.Color)
		color = &c
	}
	message := types.Message{
		URI:       URI(event),
		CID:       event.Commit.CID,
		DID:       event.Did,
		SignetURI: mr.SignetURI,
		Body:      mr.Body,
		Nick:      mr.Nick,
		Color:     color,
		PostedAt:  then,
	}
	return &message, nil
}

func (h *handler) handleSignet(ctx context.Context, event *models.Event) error {
	h.l.Deprintln("handling signet")
	switch event.Commit.Operation {
	case "create":
		return h.handleSignetCreate(ctx, event)
	case "update":
		return h.handleSignetUpdate(ctx, event)
	case "delete":
		return h.handleSignetDelete(ctx, event)
	}
	return errors.New("unimplemented Operation")
}

func (h *handler) handleSignetCreate(ctx context.Context, event *models.Event) error {
	signet, err := parseSignetRecord(event)
	if err != nil {
		return errors.New("failed to parse: " + err.Error())
	}
	return h.db.StoreSignet(signet, ctx)
}

func (h *handler) handleSignetUpdate(ctx context.Context, event *models.Event) error {
	signet, err := parseSignetRecord(event)
	if err != nil {
		return errors.New("failed to parse: " + err.Error())
	}
	return h.db.UpdateSignet(signet, ctx)
}
func (h *handler) handleSignetDelete(ctx context.Context, event *models.Event) error {
	return h.db.DeleteSignet(URI(event), ctx)
}

func parseSignetRecord(event *models.Event) (*types.Signet, error) {
	var sr lex.SignetRecord
	err := json.Unmarshal(event.Commit.Record, &sr)
	if err != nil {
		return nil, errors.New("error unmarshl: " + err.Error())
	}
	var then time.Time
	if sr.StartedAt != nil {
		then, err = syntax.ParseDatetimeTime(*sr.StartedAt)
		if err != nil {
			then = time.Now()
		}
	} else {
		then = time.Now()
	}
	signet := types.Signet{
		URI:          fmt.Sprintf("at://%s/org.xcvr.feed.channel/%s", event.Did, event.Commit.RKey),
		CID:          event.Commit.CID,
		IssuerDID:    event.Did,
		AuthorHandle: sr.AuthorHandle,
		ChannelURI:   sr.ChannelURI,
		MessageID:    uint32(sr.LRCID),
		StartedAt:    then,
	}
	return &signet, nil
}

func URI(event *models.Event) string {
	return atputils.URI(event.Did, event.Commit.Collection, event.Commit.RKey)
}

func (h *handler) ensureIKnowYou(did string, ctx context.Context) error {
	_, err := h.db.ResolveDid(did, ctx)
	if err != nil {
		handle, err := atputils.TryLookupDid(ctx, did)
		if err != nil {
			return errors.New("failed to lookup previously unknown user: " + err.Error())
		}
		err = h.db.StoreDidHandle(did, handle, ctx)
		if err != nil {
			return errors.New("failed to store did_handle for a previously unknown user")
		}
	}
	return nil
}
