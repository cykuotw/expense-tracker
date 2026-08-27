package user

import (
	"database/sql"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
)

func (s *Store) UpdateOwnProfile(userID string, payload types.UpdateOwnProfilePayload) error {
	result, err := s.db.Exec(
		"UPDATE users SET firstname = $1, lastname = $2, nickname = $3 WHERE id = $4 AND is_active = TRUE;",
		payload.Firstname,
		payload.Lastname,
		payload.Nickname,
		userID,
	)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return types.ErrUserNotExist
	}
	return nil
}

func (s *Store) ChangeOwnPassword(userID string, currentPassword string, newPassword string, preserveRefreshID string) error {
	user, err := s.GetUserByID(userID)
	if err != nil {
		return err
	}
	if !user.IsActive {
		return types.ErrAccountInactive
	}
	if user.ExternalType != "" {
		return types.ErrPasswordChangeUnavailable
	}
	if !auth.ValidatePassword(user.PasswordHashed, currentPassword) {
		return types.ErrCurrentPasswordIncorrect
	}
	if auth.ValidatePassword(user.PasswordHashed, newPassword) {
		return types.ErrPasswordUnchanged
	}

	newHash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	result, err := tx.Exec(
		"UPDATE users SET password_hash = $1 WHERE id = $2 AND password_hash = $3 AND external_type IS NULL AND is_active = TRUE;",
		newHash,
		userID,
		user.PasswordHashed,
	)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return types.ErrCurrentPasswordIncorrect
	}

	if err := revokeOtherRefreshTokens(tx, userID, preserveRefreshID); err != nil {
		return err
	}
	return tx.Commit()
}

func revokeOtherRefreshTokens(tx *sql.Tx, userID string, preserveRefreshID string) error {
	if preserveRefreshID == "" {
		_, err := tx.Exec(
			"UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND revoked_at IS NULL;",
			userID,
		)
		return err
	}
	_, err := tx.Exec(
		"UPDATE refresh_tokens SET revoked_at = NOW() WHERE user_id = $1 AND id <> $2 AND revoked_at IS NULL;",
		userID,
		preserveRefreshID,
	)
	return err
}
