package invitation_test

import (
	"database/sql"
	"expense-tracker/config"
	"expense-tracker/db"
	"expense-tracker/services/invitation"
	"expense-tracker/types"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateInvitation(t *testing.T) {
	cfg := config.Envs
	dbConn, _ := db.NewPostgreSQLStorage(cfg)
	store := invitation.NewStore(dbConn)

	// Setup inviter (User)
	inviterID := uuid.New()
	setupUser(dbConn, inviterID)
	defer cleanUser(dbConn, inviterID)

	invitationID := uuid.New()
	token := "test-token-create-" + uuid.NewString()[:8]
	inv := types.Invitation{
		ID:        invitationID,
		Token:     token,
		Email:     "invitee@test.com",
		InviterID: inviterID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}

	// Test Create
	err := store.CreateInvitation(inv)
	assert.Nil(t, err)
	defer cleanInvitation(dbConn, invitationID)

	// Verify in DB
	savedInv, err := store.GetInvitationByToken(token)
	assert.Nil(t, err)
	assert.Equal(t, inv.ID, savedInv.ID)
	assert.Equal(t, inv.Email, savedInv.Email)
}

func TestGetInvitationByToken(t *testing.T) {
	cfg := config.Envs
	dbConn, _ := db.NewPostgreSQLStorage(cfg)
	store := invitation.NewStore(dbConn)

	inviterID := uuid.New()
	setupUser(dbConn, inviterID)
	defer cleanUser(dbConn, inviterID)

	invitationID := uuid.New()
	token := "test-token-get-" + uuid.NewString()[:8]
	inv := types.Invitation{
		ID:        invitationID,
		Token:     token,
		Email:     "get@test.com",
		InviterID: inviterID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	insertInvitation(dbConn, inv)
	defer cleanInvitation(dbConn, invitationID)

	// Test Valid
	res, err := store.GetInvitationByToken(token)
	assert.Nil(t, err)
	assert.Equal(t, inv.ID, res.ID)

	// Test Invalid
	_, err = store.GetInvitationByToken("invalid-token-" + uuid.NewString())
	assert.Error(t, err)
}

func TestMarkInvitationUsed(t *testing.T) {
	cfg := config.Envs
	dbConn, _ := db.NewPostgreSQLStorage(cfg)
	store := invitation.NewStore(dbConn)

	inviterID := uuid.New()
	setupUser(dbConn, inviterID)
	defer cleanUser(dbConn, inviterID)

	invitationID := uuid.New()
	token := "test-token-mark-" + uuid.NewString()[:8]
	inv := types.Invitation{
		ID:        invitationID,
		Token:     token,
		Email:     "mark@test.com",
		InviterID: inviterID,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
	}
	insertInvitation(dbConn, inv)
	defer cleanInvitation(dbConn, invitationID)

	// Mark as used
	err := store.MarkInvitationUsed(token)
	assert.Nil(t, err)

	// Verify
	updatedInv, err := store.GetInvitationByToken(token)
	assert.Nil(t, err)
	assert.NotNil(t, updatedInv.UsedAt)
}

// Helpers

func setupUser(db *sql.DB, id uuid.UUID) {
	// We need a user to satisfy the foreign key constraint on invitations.inviter_id
	query := fmt.Sprintf(`INSERT INTO users (id, username, firstname, lastname, nickname, email, password_hash, create_time_utc, is_active, role) 
		VALUES ('%s', 'testuser_%s', 'Test', 'User', 'test', 'test_%s@test.com', 'hash', '%s', true, 'admin')`,
		id, id.String()[:8], id.String()[:8], time.Now().Format("2006-01-02 15:04:05-0700"))
	db.Exec(query)
}

func cleanUser(db *sql.DB, id uuid.UUID) {
	db.Exec(fmt.Sprintf("DELETE FROM users WHERE id = '%s'", id))
}

func insertInvitation(db *sql.DB, inv types.Invitation) {
	query := fmt.Sprintf(
		"INSERT INTO invitations (id, token, email, inviter_id, expires_at, created_at) VALUES ('%s', '%s', '%s', '%s', '%s', '%s');",
		inv.ID, inv.Token, inv.Email, inv.InviterID,
		inv.ExpiresAt.Format("2006-01-02 15:04:05"),
		inv.CreatedAt.Format("2006-01-02 15:04:05"),
	)
	db.Exec(query)
}

func cleanInvitation(db *sql.DB, id uuid.UUID) {
	db.Exec(fmt.Sprintf("DELETE FROM invitations WHERE id = '%s'", id))
}
