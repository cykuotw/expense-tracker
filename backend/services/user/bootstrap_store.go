package user

import (
	"context"
	"database/sql"
	"errors"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"fmt"

	"github.com/google/uuid"
)

const bootstrapAdminLockID int64 = 726814994

func (s *Store) ReconcileFirstAdmin(candidate *types.User, normalizedEmail string) (BootstrapStatus, error) {
	// The advisory lock provides serialization. Read committed ensures a caller
	// that waited for the lock sees the preceding bootstrap transaction's commit.
	tx, err := s.db.BeginTx(context.Background(), &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return "", err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec("SELECT pg_advisory_xact_lock($1);", bootstrapAdminLockID); err != nil {
		return "", err
	}

	var protectedEmail string
	var protectedRole sql.NullString
	var protectedActive bool
	err = tx.QueryRow(`
		SELECT email, role, is_active
		FROM users
		WHERE is_protected_admin IS TRUE
		FOR UPDATE;
	`).Scan(&protectedEmail, &protectedRole, &protectedActive)
	if err == nil {
		if !protectedRole.Valid || protectedRole.String != "admin" || !protectedActive {
			return "", errors.New("protected administrator database invariant is invalid")
		}
		if candidate != nil && auth.NormalizeEmail(protectedEmail) != normalizedEmail {
			return "", errors.New("configured first administrator does not match the protected administrator; restore the original FIRST_ADMIN_EMAIL")
		}
		if err := tx.Commit(); err != nil {
			return "", err
		}
		return BootstrapStatusAlreadyExists, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	if candidate == nil {
		var adminExists bool
		if err := tx.QueryRow("SELECT EXISTS (SELECT 1 FROM users WHERE role = 'admin');").Scan(&adminExists); err != nil {
			return "", err
		}
		if adminExists {
			return "", errors.New("first administrator reconciliation is required; configure FIRST_ADMIN_EMAIL for an existing active administrator")
		}
		if err := tx.Commit(); err != nil {
			return "", err
		}
		return BootstrapStatusNotRequested, nil
	}

	var matchedID uuid.UUID
	var matchedRole sql.NullString
	var matchedActive bool
	err = tx.QueryRow(`
		SELECT id, role, is_active
		FROM users
		WHERE LOWER(BTRIM(email)) = $1
		FOR UPDATE;
	`, normalizedEmail).Scan(&matchedID, &matchedRole, &matchedActive)
	if err == nil {
		if !matchedRole.Valid || matchedRole.String != "admin" || !matchedActive {
			return "", errors.New("configured first administrator exists but is not an active administrator")
		}
		result, err := tx.Exec("UPDATE users SET is_protected_admin = TRUE WHERE id = $1;", matchedID)
		if err != nil {
			return "", err
		}
		if err := requireOneAffected(result, "reconciled protected administrator"); err != nil {
			return "", err
		}
		if err := tx.Commit(); err != nil {
			return "", err
		}
		return BootstrapStatusReconciled, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return "", err
	}

	var adminExists bool
	if err := tx.QueryRow("SELECT EXISTS (SELECT 1 FROM users WHERE role = 'admin');").Scan(&adminExists); err != nil {
		return "", err
	}
	if adminExists {
		return "", errors.New("configured FIRST_ADMIN_EMAIL does not match an existing administrator; refusing to create an alternative system owner")
	}

	createTime := candidate.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	result, err := tx.Exec(`
		INSERT INTO users (
			id, username, firstname, lastname, nickname, email, password_hash,
			external_type, external_id, create_time_utc, is_active, role, is_protected_admin
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, TRUE, 'admin', TRUE);
	`,
		candidate.ID, candidate.Username, candidate.Firstname, candidate.Lastname,
		candidate.Nickname, normalizedEmail, candidate.PasswordHashed,
		nullableString(candidate.ExternalType), nullableString(candidate.ExternalID), createTime,
	)
	if err != nil {
		return "", err
	}
	if err := requireOneAffected(result, "created protected administrator"); err != nil {
		return "", err
	}
	if err := tx.Commit(); err != nil {
		return "", err
	}
	return BootstrapStatusCreated, nil
}

func requireOneAffected(result sql.Result, action string) error {
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return fmt.Errorf("unexpected row count for %s: %d", action, affected)
	}
	return nil
}
