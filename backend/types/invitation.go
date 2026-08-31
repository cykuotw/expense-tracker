package types

import (
	"time"

	"github.com/google/uuid"
)

type Invitation struct {
	ID        uuid.UUID  `json:"id"`
	TokenHash string     `json:"-"`
	Email     string     `json:"email"`
	InviterID uuid.UUID  `json:"inviterId"`
	ExpiresAt time.Time  `json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

type InvitationStore interface {
	CreateInvitation(invitation Invitation) error
	ExchangeInvitation(token string, registrationSession string) (*Invitation, error)
}

type CreateInvitationPayload struct {
	Email string `json:"email"`
}

type InvitationResponse struct {
	Email string `json:"email"`
	Valid bool   `json:"valid"`
}

type AdminInvitationResponse struct {
	ID        uuid.UUID  `json:"id"`
	Email     string     `json:"email"`
	ExpiresAt time.Time  `json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt"`
	CreatedAt time.Time  `json:"createdAt"`
	Status    string     `json:"status"`
}
