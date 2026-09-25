package user

import (
	"context"
	"database/sql"
	"errors"

	"expense-tracker/backend/types"
)

// HasReceiptOCRGrant reports the effective grant for an existing user. An
// inactive account never has an effective grant, even if its stored grant has
// not been removed yet.
func (s *Store) HasReceiptOCRGrant(ctx context.Context, userID string) (bool, error) {
	var granted bool
	err := s.db.QueryRowContext(ctx, `
		SELECT users.is_active AND EXISTS (
			SELECT 1
			FROM user_feature_grant
			WHERE user_feature_grant.user_id = users.id
			  AND user_feature_grant.feature_key = $2
		)
		FROM users
		WHERE users.id = $1`, userID, types.UserFeatureReceiptOCR).Scan(&granted)
	if errors.Is(err, sql.ErrNoRows) {
		return false, types.ErrUserNotExist
	}
	if err != nil {
		return false, err
	}
	return granted, nil
}

// SetReceiptOCRGrant changes the stored grant after independently verifying
// the actor is still an active administrator. Enabling a grant for an inactive
// account is rejected; revocation remains available for cleanup.
func (s *Store) SetReceiptOCRGrant(ctx context.Context, actorID, targetID string, enabled bool) error {
	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, "SELECT pg_advisory_xact_lock($1)", adminStateLockID); err != nil {
		return err
	}

	var actorRole string
	var actorActive bool
	err = tx.QueryRowContext(ctx,
		"SELECT role, is_active FROM users WHERE id = $1 FOR UPDATE", actorID,
	).Scan(&actorRole, &actorActive)
	if errors.Is(err, sql.ErrNoRows) {
		return types.ErrPermissionDenied
	}
	if err != nil {
		return err
	}
	if !actorActive {
		return types.ErrAccountInactive
	}
	if actorRole != "admin" {
		return types.ErrPermissionDenied
	}

	var targetActive bool
	err = tx.QueryRowContext(ctx,
		"SELECT is_active FROM users WHERE id = $1 FOR UPDATE", targetID,
	).Scan(&targetActive)
	if errors.Is(err, sql.ErrNoRows) {
		return types.ErrUserNotExist
	}
	if err != nil {
		return err
	}
	if enabled && !targetActive {
		return types.ErrAccountInactive
	}

	if enabled {
		_, err = tx.ExecContext(ctx, `
			INSERT INTO user_feature_grant (
				user_id, feature_key, granted_by_user_id, granted_at
			) VALUES ($1, $2, $3, NOW())
			ON CONFLICT (user_id, feature_key) DO UPDATE
			SET granted_by_user_id = EXCLUDED.granted_by_user_id,
				granted_at = EXCLUDED.granted_at`,
			targetID, types.UserFeatureReceiptOCR, actorID)
	} else {
		_, err = tx.ExecContext(ctx, `
			DELETE FROM user_feature_grant
			WHERE user_id = $1 AND feature_key = $2`, targetID, types.UserFeatureReceiptOCR)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}
