package types

import (
	"time"

	"github.com/google/uuid"
)

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	RevokedAt *time.Time
	CreatedAt time.Time
}

type RefreshTokenStore interface {
	CreateRefreshToken(token RefreshToken) error
	GetRefreshTokenByID(id string) (*RefreshToken, error)
	RevokeRefreshToken(id string) error
}
