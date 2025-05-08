package types

import "time"

type Profile struct {
	DID         string
	DisplayName string
	DefaultNick string
	Status      *string
	AvatarCID   *string
	AvatarMIME  *string
	Color       uint32
	URI         string
	CID         string
	IndexedAt   time.Time
}

type ProfileView struct {
	DID         string  `json:"did"`
	Handle      string  `json:"handle"`
	DisplayName *string `json:"displayName,omitempty"`
	Status      *string `json:"status,omitempty"`
	Color       *uint32 `json:"color,omitempty"`
	Avatar      *string `json:"avatar,omitempty"`
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

type GetChannelRequest struct {
	Limit  *int    `json:"limit,omitempty"`
	Cursor *string `json:"cursor,omitempty"`
}

type ChannelView struct {
	URI            string      `json:"uri"`
	Host           string      `json:"host"`
	Creator        ProfileView `json:"creator"`
	Title          string      `json:"title"`
	ConnectedCount *int        `json:"int"`
	Topic          *string     `json:"topic"`
	CreatedAt      time.Time   `json:"createdAt"`
}

type Signet struct {
	URI        string
	DID        string
	ChannelURI string
	MessageID  uint32
	CID        string
	StartedAt  time.Time
	IndexedAt  time.Time
}

type Message struct {
	URI       string
	DID       string
	SignetURI string
	Body      string
	Nick      string
	Color     uint32
	CID       string
	PostedAt  time.Time
	IndexedAt time.Time
}

type MessageView struct {
	URI       string      `json:"uri"`
	Author    ProfileView `json:"author"`
	Body      string      `json:"body"`
	Nick      string      `json:"nick,omitempty"`
	Color     int         `json:"color"`
	StartedAt time.Time   `json:"startedAt"`
	PostedAt  time.Time   `json:"postedAt"`
}
