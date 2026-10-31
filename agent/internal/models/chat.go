package models

import (
	"regexp"
	"strings"
	"time"
	"unicode/utf8"
)

const MaxTextBytes = 8192

var identifier = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

func ValidID(id string) bool { return identifier.MatchString(id) }

type ChatRequest struct {
	ConversationID string `json:"conversationId"`
	RunID          string `json:"runId,omitempty"`
	MessageID      string `json:"messageId"`
	Text           string `json:"text"`
}

func (request ChatRequest) Validate() error {
	if !ValidID(request.ConversationID) || !ValidID(request.MessageID) || (request.RunID != "" && !ValidID(request.RunID)) {
		return &Error{Code: Validation}
	}
	if !utf8.ValidString(request.Text) || strings.TrimSpace(request.Text) == "" || len(request.Text) > MaxTextBytes {
		return &Error{Code: Validation}
	}
	return nil
}

// Output carries visible chat events and correlation metadata, never raw ADK content.
type Output struct {
	Kind                 string    `json:"kind"`
	RunID                string    `json:"runId"`
	ConversationID       string    `json:"conversationId"`
	ClientMessageID      string    `json:"clientMessageId"`
	MessageID            string    `json:"messageId"`
	EventID              string    `json:"eventId"`
	CreatedAt            time.Time `json:"createdAt,omitzero"`
	InvocationID         string    `json:"invocationId"`
	Skill                string    `json:"skill"`
	SkillVersion         string    `json:"skillVersion"`
	Model                string    `json:"model"`
	ProviderModelVersion string    `json:"providerModelVersion"`
	Delta                string    `json:"delta"`
	ErrorCode            string    `json:"errorCode"`
}
