package auth

import (
	"context"
	"database/sql"
	"errors"
	"expense-tracker/backend/types"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgconn"
)

type RegistrationStore struct {
	db *sql.DB
}

func NewRegistrationStore(db *sql.DB) *RegistrationStore {
	return &RegistrationStore{db: db}
}

func (s *RegistrationStore) CreateInvitedUser(ctx context.Context, token string, user types.User) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("registration store is unavailable")
	}

	token = strings.TrimSpace(token)
	if token == "" {
		return types.ErrInvitationRequired
	}
	user.Email = NormalizeEmail(user.Email)

	tx, err := s.db.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelReadCommitted})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var invitationEmail string
	var used bool
	var expired bool
	err = tx.QueryRow(`
		SELECT email, used_at IS NOT NULL, expires_at <= NOW()
		FROM invitations
		WHERE token = $1
		FOR UPDATE`, token).Scan(&invitationEmail, &used, &expired)
	if errors.Is(err, sql.ErrNoRows) {
		return types.ErrInvitationInvalid
	}
	if err != nil {
		return err
	}
	if used {
		return types.ErrInvitationUsed
	}
	if expired {
		return types.ErrInvitationExpired
	}
	if normalizedInvitationEmail := NormalizeEmail(invitationEmail); normalizedInvitationEmail != "" && normalizedInvitationEmail != user.Email {
		return types.ErrInvitationEmailMismatch
	}

	_, err = tx.Exec(`
		INSERT INTO users (
			id, username, firstname, lastname, nickname, email, password_hash,
			has_local_password, external_type, external_id, create_time_utc, is_active, role
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NULLIF(BTRIM($9), ''), NULLIF(BTRIM($10), ''), $11, $12, $13)`,
		user.ID, user.Username, user.Firstname, user.Lastname, user.Nickname,
		user.Email, user.PasswordHashed, user.HasLocalPassword, user.ExternalType, user.ExternalID,
		user.CreateTime.UTC(), user.IsActive, user.Role)
	if err != nil {
		if isUniqueViolation(err) {
			return types.ErrAccountConflict
		}
		return err
	}

	result, err := tx.Exec(`
		UPDATE invitations
		SET used_at = NOW(), email = $2
		WHERE token = $1 AND used_at IS NULL`, token, user.Email)
	if err != nil {
		return err
	}
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected != 1 {
		return types.ErrInvitationUsed
	}

	return tx.Commit()
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}
