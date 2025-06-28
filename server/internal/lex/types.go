package lex

import (
	"github.com/bluesky-social/indigo/lex/util"
)
func init() {
	util.RegisterType("org.xcvr.actor.profile", &ProfileRecord{})
}

type ProfileRecord struct {
	LexiconTypeID string        `json:"$type,const=org.xcvr.actor.profile" cborgen:"$type,const=org.xcvr.actor.profile"`
	DisplayName   *string       `json:"displayName,omitempty" cborgen:"displayName,omitempty"`
	DefaultNick   *string       `json:"defaultNick,omitempty" cborgen:"defaultNick,omitempty"`
	Status        *string       `json:"status,omitempty" cborgen:"status,omitempty"`
	Avatar        *util.LexBlob `json:"avatar,omitempty" cborgen:"avatar,omitempty"`
	Color         *uint64       `json:"color,omitempty" cborgen:"color,omitempty"`
}
