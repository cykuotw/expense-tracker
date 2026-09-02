package auth

import (
	"database/sql"
	"encoding/binary"
	"errors"
	"expense-tracker/backend/types"
	"time"

	"github.com/google/uuid"
)

type RefreshStore struct {
	db *sql.DB
}

func NewRefreshStore(db *sql.DB) *RefreshStore {
	return &RefreshStore{db: db}
}

func (s *RefreshStore) CreateRefreshToken(token types.RefreshToken) error {
	if token.FamilyID == uuid.Nil {
		token.FamilyID = token.ID
	}
	query := "INSERT INTO refresh_tokens (" +
		"id, family_id, user_id, token_hash, expires_at, revoked_at, created_at" +
		") VALUES ($1, $2, $3, $4, $5, $6, $7);"
	_, err := s.db.Exec(query,
		token.ID, token.FamilyID, token.UserID, token.TokenHash,
		token.ExpiresAt.UTC(), utcTimePointer(token.RevokedAt), token.CreatedAt.UTC())
	return err
}

func (s *RefreshStore) GetRefreshTokenByID(id string) (*types.RefreshToken, error) {
	query := "SELECT id, family_id, user_id, token_hash, expires_at, revoked_at, created_at FROM refresh_tokens WHERE id = $1;"
	row := s.db.QueryRow(query, id)

	token := new(types.RefreshToken)
	if err := row.Scan(
		&token.ID,
		&token.FamilyID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, types.ErrInvalidToken
		}
		return nil, err
	}

	if token.ID == uuid.Nil {
		return nil, types.ErrInvalidToken
	}
	token.ExpiresAt = token.ExpiresAt.UTC()
	token.CreatedAt = token.CreatedAt.UTC()
	token.RevokedAt = utcTimePointer(token.RevokedAt)

	return token, nil
}

func utcTimePointer(value *time.Time) *time.Time {
	if value == nil {
		return nil
	}
	utc := value.UTC()
	return &utc
}

func (s *RefreshStore) RotateRefreshToken(id string, tokenHash string, successor types.RefreshToken) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var storedHash string
	var familyID uuid.UUID
	var userID uuid.UUID
	var revokedAt *time.Time
	var unexpired bool
	err = tx.QueryRow(
		`SELECT token_hash, family_id, user_id, revoked_at, expires_at > NOW()
		 FROM refresh_tokens
		 WHERE id = $1;`,
		id,
	).Scan(&storedHash, &familyID, &userID, &revokedAt, &unexpired)
	if errors.Is(err, sql.ErrNoRows) {
		return types.ErrInvalidToken
	}
	if err != nil {
		return err
	}
	if !tokenHashesEqual(storedHash, tokenHash) || successor.UserID != userID || successor.ID == uuid.Nil {
		return types.ErrInvalidToken
	}
	if err := lockRefreshTokenFamily(tx, familyID); err != nil {
		return err
	}

	// A request that observed the predecessor as active competes for the
	// conditional update below. Only a request that observed it as already
	// revoked is treated as reuse, so overlapping refreshes cannot create two
	// successors or revoke the winner merely because they raced.
	if revokedAt != nil {
		if err := revokeRefreshTokenFamily(tx, familyID, userID); err != nil {
			return err
		}
		if err := tx.Commit(); err != nil {
			return err
		}
		return types.ErrInvalidToken
	}
	if !unexpired {
		return types.ErrInvalidToken
	}

	result, err := tx.Exec(
		`UPDATE refresh_tokens
		 SET revoked_at = NOW()
		 WHERE id = $1
		   AND token_hash = $2
		   AND revoked_at IS NULL
		   AND expires_at > NOW();`,
		id,
		tokenHash,
	)
	if err != nil {
		return err
	}
	affected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if affected != 1 {
		return types.ErrInvalidToken
	}

	_, err = tx.Exec(
		`INSERT INTO refresh_tokens (
			id, family_id, user_id, token_hash, expires_at, revoked_at, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7);`,
		successor.ID,
		familyID,
		userID,
		successor.TokenHash,
		successor.ExpiresAt.UTC(),
		utcTimePointer(successor.RevokedAt),
		successor.CreatedAt.UTC(),
	)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (s *RefreshStore) RevokeRefreshTokenFamily(id string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() {
		_ = tx.Rollback()
	}()

	var familyID uuid.UUID
	var userID uuid.UUID
	err = tx.QueryRow(
		"SELECT family_id, user_id FROM refresh_tokens WHERE id = $1;",
		id,
	).Scan(&familyID, &userID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if err := lockRefreshTokenFamily(tx, familyID); err != nil {
		return err
	}
	if err := revokeRefreshTokenFamily(tx, familyID, userID); err != nil {
		return err
	}
	return tx.Commit()
}

func lockRefreshTokenFamily(tx *sql.Tx, familyID uuid.UUID) error {
	lockID := int64(binary.BigEndian.Uint64(familyID[:8]))
	_, err := tx.Exec("SELECT pg_advisory_xact_lock($1);", lockID)
	return err
}

func revokeRefreshTokenFamily(tx *sql.Tx, familyID uuid.UUID, userID uuid.UUID) error {
	_, err := tx.Exec(
		`UPDATE refresh_tokens
		 SET revoked_at = NOW()
		 WHERE family_id = $1
		   AND user_id = $2
		   AND revoked_at IS NULL;`,
		familyID,
		userID,
	)
	return err
}
