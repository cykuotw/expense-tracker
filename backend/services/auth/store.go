package auth

import (
	"database/sql"
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
	query := "INSERT INTO refresh_tokens (" +
		"id, user_id, token_hash, expires_at, revoked_at, created_at" +
		") VALUES ($1, $2, $3, $4, $5, $6);"
	_, err := s.db.Exec(query,
		token.ID, token.UserID, token.TokenHash,
		token.ExpiresAt.UTC(), utcTimePointer(token.RevokedAt), token.CreatedAt.UTC())
	return err
}

func (s *RefreshStore) GetRefreshTokenByID(id string) (*types.RefreshToken, error) {
	query := "SELECT id, user_id, token_hash, expires_at, revoked_at, created_at FROM refresh_tokens WHERE id = $1;"
	row := s.db.QueryRow(query, id)

	token := new(types.RefreshToken)
	if err := row.Scan(
		&token.ID,
		&token.UserID,
		&token.TokenHash,
		&token.ExpiresAt,
		&token.RevokedAt,
		&token.CreatedAt,
	); err != nil {
		if err == sql.ErrNoRows {
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

func (s *RefreshStore) RevokeRefreshToken(id string) error {
	query := "UPDATE refresh_tokens SET revoked_at = NOW() WHERE id = $1;"
	_, err := s.db.Exec(query, id)
	return err
}
