package types

import (
	"time"

	"github.com/google/uuid"
)

type ExpenseReceipt struct {
	ID               uuid.UUID
	ExpenseID        uuid.UUID
	AccountID        uuid.UUID
	ObjectID         uuid.UUID
	TemporaryKey     string
	RetainedKey      string
	RequestKey       uuid.UUID
	TokenSHA256      string
	DeleteRequestKey *uuid.UUID
	Role             string
	Status           string
	ContentType      string
	ByteSize         int64
	ChecksumSHA256   string
	Width            int
	Height           int
	SourceExpiresAt  time.Time
	AttemptCount     int
	LastErrorCode    string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	FinalizedAt      *time.Time
	DeletedAt        *time.Time
}

type ExpenseReceiptSummary struct {
	Status      string `json:"status"`
	ContentType string `json:"contentType"`
	ByteSize    int64  `json:"byteSize"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

type ReceiptChoice struct {
	Keep  bool   `json:"keep"`
	Token string `json:"token,omitempty"`
}
