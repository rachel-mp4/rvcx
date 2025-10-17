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
		"org.xcvr.lrc.media",
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
	case "org.xcvr.lrc.media":
		return h.handleMedia(ctx, event)
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
		h.l.Println("error unmarshaling: " + err.Error())
		return nil
	}
	err = h.rm.AcceptProfile(pr, event.Did, ctx)
	if err != nil {
		h.l.Println(err.Error())
	}
	return nil
}

func (h *handler) handleProfileDelete(ctx context.Context, event *models.Event) error {
	err := h.rm.DeleteProfile(event.Did, event.Commit.CID, ctx)
	if err != nil {
		h.l.Println(err.Error())
	}
	return nil
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
		h.l.Println("i couldn't create the channel: " + err.Error())
		return nil
	}
	err = h.rm.AcceptChannel(channel, ctx)
	if err != nil {
		h.l.Println(err.Error())
	}
	return nil
}

func (h *handler) handleChannelUpdate(ctx context.Context, event *models.Event) error {
	channel, err := parseChannelRecord(event)
	if err != nil {
		h.l.Println("i couldn't create the channel: " + err.Error())
		return nil
	}
	err = h.rm.AcceptChannelUpdate(channel, ctx)
	if err != nil {
		h.l.Println(err.Error())
	}
	return nil
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
	err := h.rm.AcceptChannelDelete(URI(event), ctx)
	if err != nil {
		h.l.Println(err.Error())
	}
	return nil
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
		h.l.Println("error parsing: " + err.Error())
		return nil
	}
	err = h.rm.AcceptMessage(message, ctx)
	if err != nil {
		h.l.Println(err.Error())
	}
	return nil
}

func (h *handler) handleMessageUpdate(ctx context.Context, event *models.Event) error {
	message, err := parseMessageRecord(event)
	if err != nil {
		h.l.Println("error parsing: " + err.Error())
	}
	err = h.rm.AcceptMessageUpdate(message, event.Did, ctx)
	if err != nil {
		h.l.Println(err.Error())
	}
	return nil
}

func (h *handler) handleMessageDelete(ctx context.Context, event *models.Event) error {
	err := h.rm.AcceptMessageDelete(URI(event), ctx)
	if err != nil {
		h.l.Println(err.Error())
	}
	return nil
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
		h.l.Println("failed to parse: " + err.Error())
		return nil
	}
	err = h.rm.AcceptSignet(signet, ctx)
	if err != nil {
		h.l.Println(err.Error())
	}
	return nil
}

func (h *handler) handleSignetUpdate(ctx context.Context, event *models.Event) error {
	signet, err := parseSignetRecord(event)
	if err != nil {
		h.l.Println("failed to parse: " + err.Error())
		return nil
	}
	err = h.rm.AcceptSignetUpdate(signet, ctx)
	if err != nil {
		h.l.Println(err.Error())
	}
	return nil
}
func (h *handler) handleSignetDelete(ctx context.Context, event *models.Event) error {
	err := h.rm.AcceptSignetDelete(URI(event), ctx)
	if err != nil {
		h.l.Println(err.Error())
	}
	return nil
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
		URI:          fmt.Sprintf("at://%s/org.xcvr.lrc.signet/%s", event.Did, event.Commit.RKey),
		CID:          event.Commit.CID,
		IssuerDID:    event.Did,
		AuthorHandle: sr.AuthorHandle,
		ChannelURI:   sr.ChannelURI,
		MessageID:    uint32(sr.LRCID),
		StartedAt:    then,
	}
	return &signet, nil
}

func (h *handler) handleMedia(ctx context.Context, event *models.Event) error {
	h.l.Deprintln("handling media")
	switch event.Commit.Operation {
	case "create":
		return h.handleMediaCreate(ctx, event)
	case "update":
		return h.handleMediaUpdate(ctx, event)
	case "delete":
		return h.handleMediaDelete(ctx, event)
	}
	return errors.New("unimplemented Operation")
}

func (h *handler) handleMediaCreate(ctx context.Context, event *models.Event) error {
	mr, err := parseMediaRecord(event)
	if err != nil {
		h.l.Deprintln(err.Error())
		return nil
	}
	if mr.Image != nil {
		image, err := wrangeMediaRecordIntoImage(event, mr)
		if err != nil {
			h.l.Deprintln(err.Error())
			return nil
		}
		err = h.rm.AcceptImage(image, ctx)
		if err != nil {
			h.l.Deprintln(err.Error())
			return nil
		}
		return nil
	}
	return nil
}

func (h *handler) handleMediaUpdate(ctx context.Context, event *models.Event) error {
	mr, err := parseMediaRecord(event)
	if err != nil {
		h.l.Deprintln(err.Error())
		return nil
	}
	if mr.Image != nil {
		image, err := wrangeMediaRecordIntoImage(event, mr)
		if err != nil {
			h.l.Deprintln(err.Error())
			return nil
		}
		err = h.rm.AcceptImageUpdate(image, ctx)
		if err != nil {
			h.l.Deprintln(err.Error())
			return nil
		}
		return nil
	}
	return nil
}

func (h *handler) handleMediaDelete(ctx context.Context, event *models.Event) error {
	mr, err := parseMediaRecord(event)
	if err != nil {
		h.l.Deprintln(err.Error())
		return nil
	}
	if mr.Image != nil {
		image, err := wrangeMediaRecordIntoImage(event, mr)
		if err != nil {
			h.l.Deprintln(err.Error())
			return nil
		}
		err = h.rm.AcceptImageDelete(image, ctx)
		if err != nil {
			h.l.Deprintln(err.Error())
			return nil
		}
		return nil
	}
	return nil
}

func parseMediaRecord(event *models.Event) (*lex.MediaRecord, error) {
	var mr lex.MediaRecord
	err := json.Unmarshal(event.Commit.Record, &mr)
	if err != nil {
		return nil, errors.New("error unmarshl: " + err.Error())
	}
	return &mr, nil
}

func wrangeMediaRecordIntoImage(event *models.Event, mr *lex.MediaRecord) (*types.Image, error) {
	if mr.Image != nil {
		then, err := syntax.ParseDatetimeTime(mr.PostedAt)
		if err != nil {
			then = time.Now()
		}
		var color *uint32
		if mr.Color != nil {
			c := uint32(*mr.Color)
			color = &c
		}
		var blobcid *string
		var blobmime *string
		if mr.Image.Blob != nil {
			bcid := mr.Image.Blob.Ref.String()
			bmime := mr.Image.Blob.MimeType
			blobcid = &bcid
			blobmime = &bmime
		}
		var width, height *int64
		if mr.Image.AspectRatio != nil {
			w := mr.Image.AspectRatio.Width
			h := mr.Image.AspectRatio.Height
			width = &w
			height = &h
		}
		image := types.Image{
			URI:       URI(event),
			DID:       event.Did,
			SignetURI: mr.SignetURI,
			BlobCID:   blobcid,
			BlobMIME:  blobmime,
			Alt:       mr.Image.Alt,
			Nick:      mr.Nick,
			Color:     color,
			CID:       event.Commit.CID,
			Width:     width,
			Height:    height,
			PostedAt:  then,
		}
		return &image, nil
	}
	return nil, errors.New("image should be non nil")

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
