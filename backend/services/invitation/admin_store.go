package invitation

import (
	"database/sql"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"time"
)

func (s *Store) GetAdminInvitations() ([]types.AdminInvitationResponse, error) {
	rows, err := s.db.Query(`
		SELECT id, email, expires_at, used_at, created_at
		FROM invitations
		ORDER BY created_at DESC;
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invitations := []types.AdminInvitationResponse{}
	now := time.Now()
	for rows.Next() {
		var invitation types.AdminInvitationResponse
		if err := rows.Scan(
			&invitation.ID, &invitation.Email, &invitation.ExpiresAt,
			&invitation.UsedAt, &invitation.CreatedAt,
		); err != nil {
			return nil, err
		}
		switch {
		case invitation.UsedAt != nil:
			invitation.Status = "used"
		case now.After(invitation.ExpiresAt):
			invitation.Status = "expired"
		default:
			invitation.Status = "invited"
		}
		invitations = append(invitations, invitation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return invitations, nil
}

func (s *Store) RotateInvitationTokenByID(id string) (string, error) {
	token, err := auth.GenerateOpaqueToken()
	if err != nil {
		return "", err
	}
	var expiresAt time.Time
	var usedAt sql.NullTime
	if err := s.db.QueryRow(
		"SELECT expires_at, used_at FROM invitations WHERE id = $1;",
		id,
	).Scan(&expiresAt, &usedAt); err != nil {
		return "", err
	}
	if usedAt.Valid || time.Now().After(expiresAt) {
		return "", types.ErrInvalidAction
	}
	result, err := s.db.Exec(`
		UPDATE invitations
		SET token_hash = $2, registration_session_hash = NULL, registration_session_expires_at = NULL
		WHERE id = $1 AND used_at IS NULL AND expires_at > NOW()`, id, auth.HashToken(token))
	if err != nil {
		return "", err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return "", err
	}
	if affected != 1 {
		return "", types.ErrInvalidAction
	}
	return token, nil
}

func (s *Store) ExpireInvitationByID(id string) error {
	result, err := s.db.Exec(
		"UPDATE invitations SET expires_at = NOW(), registration_session_hash = NULL, registration_session_expires_at = NULL WHERE id = $1 AND used_at IS NULL AND expires_at > NOW();",
		id,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return types.ErrInvalidAction
	}
	return nil
}
