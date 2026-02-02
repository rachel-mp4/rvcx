package types

import (
	"time"
)

type ReportRequest struct {
	URI    string `json:"uri"`
	Reason string `json:"reason"`
}

type AnonReport struct {
	Addr       string    `json:"addr"`
	URI        string    `json:"uri"`
	Reason     string    `json:"reason"`
	ReportedAt time.Time `json:"reportedAt"`
}

type AuthReport struct {
	DID        string    `json:"did"`
	URI        string    `json:"uri"`
	Reason     string    `json:"reason"`
	ReportedAt time.Time `json:"reportedAt"`
}
