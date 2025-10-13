package types

import (
	"encoding/json"
	"rvcx/internal/lex"
	"time"
)

type Profile struct {
	DID         string
	DisplayName string
	DefaultNick string
	Status      *string
	AvatarCID   *string
	AvatarMIME  *string
	Color       *uint64
	IndexedAt   time.Time
}

type PostProfileRequest struct {
	DisplayName *string `json:"displayName,omitempty"`
	Status      *string `json:"status,omitempty"`
	Color       *uint64 `json:"color,omitempty"`
	Avatar      *string `json:"avatar,omitempty"`
	DefaultNick *string `json:"defaultNick,omitempty"`
}

type ProfileView struct {
	Type        string  `json:"$type,const=org.xcvr.actor.defs#profileView"`
	DID         string  `json:"did"`
	Handle      string  `json:"handle"`
	DisplayName *string `json:"displayName,omitempty"`
	Status      *string `json:"status,omitempty"`
	Color       *uint64 `json:"color,omitempty"`
	Avatar      *string `json:"avatar,omitempty"`
	DefaultNick *string `json:"defaultNick,omitempty"`
}

func (p ProfileView) MarshalJSON() ([]byte, error) {
	type Alias ProfileView
	return json.Marshal(&struct {
		Type string `json:"$type"`
		*Alias
	}{
		Type:  "org.xcvr.actor.defs#profileView",
		Alias: (*Alias)(&p),
	})
}

type DIDHandle struct {
	Handle    string
	DID       string
	IndexedAt time.Time
}

type Channel struct {
	URI       string
	CID       string
	DID       string
	Host      string
	Title     string
	Topic     *string
	CreatedAt time.Time
	IndexedAt time.Time
}

type PostChannelRequest struct {
	Title string  `json:"title"`
	Topic *string `json:"topic,omitempty"`
	Host  string  `json:"host"`
}

type ResolveChannelRequest struct {
	DID    *string `json:"did,omitempty"`
	Handle *string `json:"handle,omitempty"`
	Rkey   string  `json:"rkey"`
}

type ResolveChannelResponse struct {
	URL string  `json:"url"`
	URI *string `json:"uri,omitempty"`
}

type GetChannelRequest struct {
	Limit  *int    `json:"limit,omitempty"`
	Cursor *string `json:"cursor,omitempty"`
}

type ChannelView struct {
	Type           string      `json:"$type,const=org.xcvr.feed.defs#channelView"`
	URI            string      `json:"uri"`
	Host           string      `json:"host"`
	Creator        ProfileView `json:"creator"`
	Title          string      `json:"title"`
	ConnectedCount *int        `json:"connectedCount,omitempty"`
	Topic          *string     `json:"topic,omitempty"`
	CreatedAt      time.Time   `json:"createdAt"`
}

func (c ChannelView) MarshalJSON() ([]byte, error) {
	type Alias ChannelView
	return json.Marshal(&struct {
		Type string `json:"$type"`
		*Alias
	}{
		Type:  "org.xcvr.feed.defs#channelView",
		Alias: (*Alias)(&c),
	})
}

type Signet struct {
	URI          string
	IssuerDID    string
	AuthorHandle string
	ChannelURI   string
	MessageID    uint32
	CID          string
	StartedAt    time.Time
	IndexedAt    time.Time
}

type SignetView struct {
	Type         string    `json:"$type,const=org.xcvr.lrc.defs#signetView"`
	URI          string    `json:"uri"`
	IssuerHandle string    `json:"issuerHandle"`
	ChannelURI   string    `json:"channelURI"`
	LrcId        uint32    `json:"lrcID"`
	AuthorHandle string    `json:"authorHandle"`
	StartedAt    time.Time `json:"startedAt"`
}

func (s SignetView) MarshalJSON() ([]byte, error) {
	type Alias SignetView
	return json.Marshal(&struct {
		Type string `json:"$type"`
		*Alias
	}{
		Type:  "org.xcvr.lrc.defs#signetView",
		Alias: (*Alias)(&s),
	})
}

type Message struct {
	URI       string
	DID       string
	SignetURI string
	Body      string
	Nick      *string
	Color     *uint32
	CID       string
	PostedAt  time.Time
	IndexedAt time.Time
}

type PostMessageRequest struct {
	SignetURI  *string `json:"signetURI,omitempty"`
	ChannelURI *string `json:"channelURI,omitempty"`
	MessageID  *uint32 `json:"messageID,omitempty"`
	Body       string  `json:"body"`
	Nick       *string `json:"nick,omitempty"`
	Color      *uint32 `json:"color,omitempty"`
	Nonce      []byte  `json:"nonce,omitempty"`
}

type MessageView struct {
	Type      string      `json:"$type,const=org.xcvr.lrc.defs#messageView"`
	URI       string      `json:"uri"`
	Author    ProfileView `json:"author"`
	Body      string      `json:"body"`
	Nick      *string     `json:"nick,omitempty"`
	Color     *uint32     `json:"color,omitempty"`
	SignetURI string      `json:"signetURI"`
	PostedAt  time.Time   `json:"postedAt"`
}

func (m MessageView) MarshalJSON() ([]byte, error) {
	type Alias MessageView
	return json.Marshal(&struct {
		Type string `json:"$type"`
		*Alias
	}{
		Type:  "org.xcvr.lrc.defs#messageView",
		Alias: (*Alias)(&m),
	})
}

type SignedMessageView struct {
	Type     string      `json:"$type,const=org.xcvr.lrc.defs#signedMessageView"`
	URI      string      `json:"uri"`
	Author   ProfileView `json:"author"`
	Body     string      `json:"body"`
	Nick     *string     `json:"nick,omitempty"`
	Color    *uint32     `json:"color,omitempty"`
	Signet   SignetView  `json:"signet"`
	PostedAt time.Time   `json:"postedAt"`
}

func (m SignedMessageView) MarshalJSON() ([]byte, error) {
	type Alias SignedMessageView
	return json.Marshal(&struct {
		Type string `json:"$type"`
		*Alias
	}{
		Type:  "org.xcvr.lrc.defs#signedMessageView",
		Alias: (*Alias)(&m),
	})
}

type GetMessagesOut struct {
	Messages []SignedMessageView `json:"messages"`
	Cursor   *string             `json:"cursor,omitempty"`
}

type Image struct {
	URI       string
	DID       string
	SignetURI string
	BlobCID   *string
	BlobMIME  *string
	Alt       string
	Nick      *string
	Color     *uint32
	CID       string
	Width     *int64
	Height    *int64
	PostedAt  time.Time
	IndexedAt time.Time
}

type MediaView struct {
	Type      string      `json:"$type,const=org.xcvr.lrc.defs#messageView"`
	URI       string      `json:"uri"`
	Author    ProfileView `json:"author"`
	Image     *ImageView  `json:"imageView,omitempty"`
	Nick      *string     `json:"nick,omitempty"`
	Color     *uint32     `json:"color,omitempty"`
	SignetURI string      `json:"signetURI"`
	PostedAt  time.Time   `json:"postedAt"`
}

type ImageView struct {
	Alt         string           `json:"alt"`
	Src         *string          `json:"src,omitempty"`
	AspectRatio *lex.AspectRatio `json:"AspectRatio,omitempty"`
}

type ParseMediaRequest struct {
	Nick      *string    `json:"nick,omitempty"`
	Color     *uint32    `json:"color,omitempty"`
	SignetURI string     `json:"signetURI"`
	Image     *lex.Image `json:"image,omitempty"`
	Type      string     `json:"type"`
}

func (m MediaView) MarshalJSON() ([]byte, error) {
	type Alias MediaView
	return json.Marshal(&struct {
		Type string `json:"$type"`
		*Alias
	}{
		Type:  "org.xcvr.lrc.defs#mediaView",
		Alias: (*Alias)(&m),
	})
}
