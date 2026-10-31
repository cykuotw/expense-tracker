// Package models defines shared agent data contracts without storage behavior.
package models

import (
	"encoding/json/jsontext"
	"time"
)

type Key struct{ App, Owner, Session string }
type UIRef struct{ ThreadID, RunID string }
type CallRef struct{ InvocationID, CallID, Agent, Branch, Path string }
type Run struct {
	Key             `json:"-"`
	UI              UIRef     `json:"-"`
	ID              string    `json:"id"`
	ClientMessageID string    `json:"clientMessageId"`
	MessageID       string    `json:"messageId"`
	Status          string    `json:"status"`
	Text            string    `json:"text"`
	ErrorCode       string    `json:"errorCode,omitempty"`
	StartedAt       time.Time `json:"startedAt"`
}
type Snapshot struct {
	CallID        string         `json:"callId,omitempty"`
	Data          jsontext.Value `json:"data"`
	Complete      bool           `json:"complete"`
	OriginalBytes int            `json:"originalBytes"`
	RetainedBytes int            `json:"retainedBytes"`
	Reason        string         `json:"reason,omitempty"`
}
type Entry struct {
	Kind         string
	Author       string
	InvocationID string
	Payload      Snapshot
}
type Conversation struct {
	ID        string    `json:"conversationId"`
	UpdatedAt time.Time `json:"updatedAt"`
}
