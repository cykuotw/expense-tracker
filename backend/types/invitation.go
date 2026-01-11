package types

import (
	"time"

	"github.com/google/uuid"
)

type Invitation struct {
	ID        uuid.UUID  `json:"id"`
	Token     string     `json:"token"`
	Email     string     `json:"email"`
	InviterID uuid.UUID  `json:"inviterId"`
	ExpiresAt time.Time  `json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt"`
	CreatedAt time.Time  `json:"createdAt"`
}

type InvitationStore interface {
	CreateInvitation(invitation Invitation) error
	GetInvitationByToken(token string) (*Invitation, error)
	MarkInvitationUsed(token string, email string) error
	ExpireInvitation(token string) error
	GetInvitations() ([]Invitation, error)
}

type CreateInvitationPayload struct {
	Email string `json:"email"`
}

type InvitationResponse struct {
	Email string `json:"email"`
	Valid bool   `json:"valid"`
}
