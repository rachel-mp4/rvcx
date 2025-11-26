package lex

import (
	"github.com/bluesky-social/indigo/lex/util"
)

func init() {
	util.RegisterType("org.xcvr.actor.profile", &ProfileRecord{})
	util.RegisterType("org.xcvr.feed.channel", &ChannelRecord{})
	util.RegisterType("org.xcvr.lrc.message", &MessageRecord{})
	util.RegisterType("org.xcvr.lrc.signet", &SignetRecord{})
}

type ProfileRecord struct {
	LexiconTypeID string        `json:"$type,const=org.xcvr.actor.profile" cborgen:"$type,const=org.xcvr.actor.profile"`
	DisplayName   *string       `json:"displayName,omitempty" cborgen:"displayName,omitempty"`
	DefaultNick   *string       `json:"defaultNick,omitempty" cborgen:"defaultNick,omitempty"`
	Status        *string       `json:"status,omitempty" cborgen:"status,omitempty"`
	Avatar        *util.LexBlob `json:"avatar,omitempty" cborgen:"avatar,omitempty"`
	Color         *uint64       `json:"color,omitempty" cborgen:"color,omitempty"`
}

type ChannelRecord struct {
	LexiconTypeID string  `json:"$type,const=org.xcvr.feed.channel" cborgen:"$type,const=org.xcvr.feed.channel"`
	Title         string  `json:"title" cborgen:"title"`
	Topic         *string `json:"topic,omitempty" cborgen:"topic,omitempty"`
	CreatedAt     string  `json:"createdAt" cborgen:"createdAt"`
	Host          string  `json:"host" cborgen:"host"`
}

type MessageRecord struct {
	LexiconTypeID string  `json:"$type,const=org.xcvr.lrc.message" cborgen:"$type,const=org.xcvr.lrc.message"`
	SignetURI     string  `json:"signetURI" cborgen:"signetURI"`
	Body          string  `json:"body" cborgen:"body"`
	Nick          *string `json:"nick,omitempty" cborgen:"nick,omitempty"`
	Color         *uint64 `json:"color,omitempty" cborgen:"color,omitempty"`
	PostedAt      string  `json:"postedAt" cborgen:"postedAt"`
}

type SignetRecord struct {
	LexiconTypeID string  `json:"$type,const=org.xcvr.lrc.signet" cborgen:"$type,const=org.xcvr.lrc.signet"`
	ChannelURI    string  `json:"channelURI" cborgen:"channelURI"`
	LRCID         uint64  `json:"lrcID" cborgen:"lrcID"`
	Author        string  `json:"author" cborgen:"author"`
	AuthorHandle  *string `json:"authorHandle,omitempty" cborgen:"authorHandle,omitempty"`
	StartedAt     *string `json:"startedAt,omitempty" cborgen:"startedAt,omitempty"`
}

type MediaRecord struct {
	LexiconTypeID string  `json:"$type,const=org.xcvr.lrc.media" cborgen:"$type,const=org.xcvr.lrc.media"`
	SignetURI     string  `json:"signetURI" cborgen:"signetURI"`
	Image         *Image  `json:"image,omitempty" cborgen:"image,omitempty"`
	Nick          *string `json:"nick,omitempty" cborgen:"nick,omitempty"`
	Color         *uint64 `json:"color,omitempty" cborgen:"color,omitempty"`
	PostedAt      string  `json:"postedAt" cborgen:"postedAt"`
}

type Image struct {
	LexiconTypeID string           `json:"$type,const=org.xcvr.lrc.image" cborgen:"$type,const=org.xcvr.lrc.image"`
	Alt           string           `json:"alt" cborgen:"alt"`
	AspectRatio   *AspectRatio     `json:"aspectRatio,omitempty" cborgen:"aspectRatio,omitempty"`
	Blob          *util.BlobSchema `json:"blob,omitempty" cborgen:"blob,omitempty"`
}

type AspectRatio struct {
	Height int64 `json:"height" cborgen:"height"`
	Width  int64 `json:"width" cborgen:"width"`
}
