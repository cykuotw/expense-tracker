package invitation

import (
	"database/sql"
	"expense-tracker/backend/types"
	"fmt"

	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateInvitation(invitation types.Invitation) error {
	query := "INSERT INTO invitations (id, token, email, inviter_id, expires_at, created_at) VALUES ($1, $2, $3, $4, $5, $6);"
	_, err := s.db.Exec(query,
		invitation.ID, invitation.Token, invitation.Email, invitation.InviterID,
		invitation.ExpiresAt.Format("2006-01-02 15:04:05"),
		invitation.CreatedAt.Format("2006-01-02 15:04:05"),
	)
	return err
}

func (s *Store) GetInvitationByToken(token string) (*types.Invitation, error) {
	query := "SELECT * FROM invitations WHERE token = $1;"
	rows, err := s.db.Query(query, token)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invitation := new(types.Invitation)
	for rows.Next() {
		err := rows.Scan(
			&invitation.ID,
			&invitation.Token,
			&invitation.Email,
			&invitation.InviterID,
			&invitation.ExpiresAt,
			&invitation.UsedAt,
			&invitation.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
	}

	if invitation.ID == uuid.Nil {
		return nil, fmt.Errorf("invitation not found")
	}

	return invitation, nil
}

func (s *Store) MarkInvitationUsed(token string, email string) error {
	query := "UPDATE invitations SET used_at = NOW(), email = $2 WHERE token = $1;"
	_, err := s.db.Exec(query, token, email)
	return err
}

func (s *Store) ExpireInvitation(token string) error {
	query := "UPDATE invitations SET expires_at = NOW() WHERE token = $1;"
	_, err := s.db.Exec(query, token)
	return err
}

func (s *Store) GetInvitations() ([]types.Invitation, error) {
	query := "SELECT * FROM invitations ORDER BY created_at DESC;"
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	invitations := []types.Invitation{}
	for rows.Next() {
		var invitation types.Invitation
		err := rows.Scan(
			&invitation.ID,
			&invitation.Token,
			&invitation.Email,
			&invitation.InviterID,
			&invitation.ExpiresAt,
			&invitation.UsedAt,
			&invitation.CreatedAt,
		)
		if err != nil {
			return nil, err
		}
		invitations = append(invitations, invitation)
	}

	return invitations, nil
}
