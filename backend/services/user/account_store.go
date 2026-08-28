package user

import (
	"context"
	"database/sql"
	"errors"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
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
	if !user.HasLocalPassword {
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
		"UPDATE users SET password_hash = $1 WHERE id = $2 AND password_hash = $3 AND has_local_password IS TRUE AND is_active = TRUE;",
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

func (s *Store) LinkGoogleIdentity(
	ctx context.Context,
	userID string,
	currentPassword string,
	externalID string,
	verifiedEmail string,
	preserveRefreshID string,
) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var passwordHash string
	var accountEmail string
	var hasLocalPassword bool
	var isActive bool
	var externalType sql.NullString
	var currentExternalID sql.NullString
	err = tx.QueryRowContext(ctx, `
		SELECT password_hash, email, has_local_password, is_active, external_type, external_id
		FROM users
		WHERE id = $1
		FOR UPDATE`, userID).Scan(
		&passwordHash,
		&accountEmail,
		&hasLocalPassword,
		&isActive,
		&externalType,
		&currentExternalID,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return types.ErrUserNotExist
	}
	if err != nil {
		return err
	}
	if !isActive {
		return types.ErrAccountInactive
	}
	if !hasLocalPassword {
		return types.ErrGoogleLinkUnavailable
	}
	if !auth.ValidatePassword(passwordHash, currentPassword) {
		return types.ErrCurrentPasswordIncorrect
	}
	if auth.NormalizeEmail(accountEmail) != auth.NormalizeEmail(verifiedEmail) {
		return types.ErrGoogleLinkEmailMismatch
	}

	externalID = strings.TrimSpace(externalID)
	if externalType.Valid || currentExternalID.Valid {
		if externalType.String == "google" && currentExternalID.String == externalID {
			return tx.Commit()
		}
		return types.ErrGoogleAlreadyConnected
	}

	result, err := tx.ExecContext(ctx, `
		UPDATE users
		SET external_type = 'google', external_id = $1
		WHERE id = $2
		  AND has_local_password IS TRUE
		  AND is_active IS TRUE
		  AND external_type IS NULL
		  AND external_id IS NULL`, externalID, userID)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return types.ErrGoogleAccountConflict
		}
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated != 1 {
		return types.ErrGoogleAlreadyConnected
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
