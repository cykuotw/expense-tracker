package invitation

import (
	"database/sql"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateInvitation(invitation types.Invitation) error {
	query := "INSERT INTO invitations (id, token_hash, email, inviter_id, expires_at, created_at) VALUES ($1, $2, $3, $4, NOW() + INTERVAL '24 hours', NOW());"
	_, err := s.db.Exec(query,
		invitation.ID, invitation.TokenHash, auth.NormalizeEmail(invitation.Email), invitation.InviterID,
	)
	return err
}

func (s *Store) ExchangeInvitation(token string, registrationSession string) (*types.Invitation, error) {
	invitation := new(types.Invitation)
	err := s.db.QueryRow(`
		UPDATE invitations
		SET registration_session_hash = $2, registration_session_expires_at = NOW() + INTERVAL '15 minutes'
		WHERE token_hash = $1 AND used_at IS NULL AND expires_at > NOW()
		RETURNING id, email, inviter_id, expires_at, used_at, created_at`,
		auth.HashToken(token), auth.HashToken(registrationSession),
	).Scan(
		&invitation.ID,
		&invitation.Email,
		&invitation.InviterID,
		&invitation.ExpiresAt,
		&invitation.UsedAt,
		&invitation.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, types.ErrInvitationInvalid
	}
	if err != nil {
		return nil, err
	}
	invitation.ExpiresAt = invitation.ExpiresAt.UTC()
	invitation.CreatedAt = invitation.CreatedAt.UTC()
	if invitation.UsedAt != nil {
		usedAt := invitation.UsedAt.UTC()
		invitation.UsedAt = &usedAt
	}
	return invitation, nil
}
