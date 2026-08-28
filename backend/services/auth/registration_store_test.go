package auth_test

import (
	"database/sql"
	"expense-tracker/backend/config"
	"expense-tracker/backend/db"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func openRegistrationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	conn, err := db.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		t.Skipf("skipping: db connect error: %v", err)
	}
	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		t.Skipf("skipping: db ping error: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}

func TestRegistrationStoreConsumesInvitationExactlyOnce(t *testing.T) {
	conn := openRegistrationTestDB(t)
	inviterID := insertRegistrationUser(t, conn, "inviter-"+uuid.NewString()+"@example.test")
	t.Cleanup(func() { _, _ = conn.Exec("DELETE FROM users WHERE id = $1", inviterID) })

	invitationID := uuid.New()
	token := "registration-concurrency-" + uuid.NewString()
	_, err := conn.Exec(`INSERT INTO invitations (id, token, email, inviter_id, expires_at, created_at)
		VALUES ($1, $2, '', $3, NOW() + INTERVAL '1 hour', NOW())`, invitationID, token, inviterID)
	require.NoError(t, err)

	store := auth.NewRegistrationStore(conn)
	users := []types.User{
		newRegistrationUser("first-" + uuid.NewString() + "@example.test"),
		newRegistrationUser("second-" + uuid.NewString() + "@example.test"),
	}
	for _, user := range users {
		userID := user.ID
		t.Cleanup(func() { _, _ = conn.Exec("DELETE FROM users WHERE id = $1", userID) })
	}

	start := make(chan struct{})
	results := make(chan error, len(users))
	var wg sync.WaitGroup
	for _, user := range users {
		wg.Add(1)
		go func(candidate types.User) {
			defer wg.Done()
			<-start
			results <- store.CreateInvitedUser(t.Context(), token, candidate)
		}(user)
	}
	close(start)
	wg.Wait()
	close(results)

	successes, usedErrors := 0, 0
	for result := range results {
		switch {
		case result == nil:
			successes++
		case assert.ErrorIs(t, result, types.ErrInvitationUsed):
			usedErrors++
		}
	}
	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, usedErrors)

	var createdCount int
	require.NoError(t, conn.QueryRow("SELECT COUNT(*) FROM users WHERE id = ANY($1)", []uuid.UUID{users[0].ID, users[1].ID}).Scan(&createdCount))
	assert.Equal(t, 1, createdCount)
}

func TestRegistrationStoreConflictRollsBackInvitationConsumption(t *testing.T) {
	conn := openRegistrationTestDB(t)
	existingID := insertRegistrationUser(t, conn, "existing-"+uuid.NewString()+"@example.test")
	t.Cleanup(func() { _, _ = conn.Exec("DELETE FROM users WHERE id = $1", existingID) })

	invitationID := uuid.New()
	token := "registration-rollback-" + uuid.NewString()
	_, err := conn.Exec(`INSERT INTO invitations (id, token, email, inviter_id, expires_at, created_at)
		VALUES ($1, $2, '', $3, NOW() + INTERVAL '1 hour', NOW())`, invitationID, token, existingID)
	require.NoError(t, err)

	conflicting := newRegistrationUser("different-" + uuid.NewString() + "@example.test")
	conflicting.ID = existingID
	err = auth.NewRegistrationStore(conn).CreateInvitedUser(t.Context(), token, conflicting)
	assert.ErrorIs(t, err, types.ErrAccountConflict)

	var usedAt sql.NullTime
	require.NoError(t, conn.QueryRow("SELECT used_at FROM invitations WHERE id = $1", invitationID).Scan(&usedAt))
	assert.False(t, usedAt.Valid)
}

func TestRegistrationStoreNormalizedEmailUniqueAcrossInvitations(t *testing.T) {
	conn := openRegistrationTestDB(t)
	inviterID := insertRegistrationUser(t, conn, "inviter-"+uuid.NewString()+"@example.test")
	t.Cleanup(func() { _, _ = conn.Exec("DELETE FROM users WHERE id = $1", inviterID) })

	tokens := []string{"normalized-first-" + uuid.NewString(), "normalized-second-" + uuid.NewString()}
	invitationIDs := []uuid.UUID{uuid.New(), uuid.New()}
	for index := range tokens {
		_, err := conn.Exec(`INSERT INTO invitations (id, token, email, inviter_id, expires_at, created_at)
			VALUES ($1, $2, '', $3, NOW() + INTERVAL '1 hour', NOW())`, invitationIDs[index], tokens[index], inviterID)
		require.NoError(t, err)
	}

	address := "same-" + uuid.NewString() + "@example.test"
	users := []types.User{newRegistrationUser("  " + address + "  "), newRegistrationUser(strings.ToUpper(address))}
	for _, user := range users {
		userID := user.ID
		t.Cleanup(func() { _, _ = conn.Exec("DELETE FROM users WHERE id = $1", userID) })
	}

	store := auth.NewRegistrationStore(conn)
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for index := range users {
		wg.Add(1)
		go func(candidate types.User, token string) {
			defer wg.Done()
			<-start
			results <- store.CreateInvitedUser(t.Context(), token, candidate)
		}(users[index], tokens[index])
	}
	close(start)
	wg.Wait()
	close(results)

	successes, conflicts := 0, 0
	for result := range results {
		if result == nil {
			successes++
		} else if assert.ErrorIs(t, result, types.ErrAccountConflict) {
			conflicts++
		}
	}
	assert.Equal(t, 1, successes)
	assert.Equal(t, 1, conflicts)

	var usedInvitations int
	require.NoError(t, conn.QueryRow("SELECT COUNT(*) FROM invitations WHERE id = ANY($1) AND used_at IS NOT NULL", invitationIDs).Scan(&usedInvitations))
	assert.Equal(t, 1, usedInvitations)
}

func insertRegistrationUser(t *testing.T, conn *sql.DB, email string) uuid.UUID {
	t.Helper()
	user := newRegistrationUser(email)
	_, err := conn.Exec(`INSERT INTO users (
		id, username, firstname, lastname, nickname, email, password_hash,
		create_time_utc, is_active, role
	) VALUES ($1, $2, 'Test', 'User', $2, $3, 'hash', $4, true, 'admin')`,
		user.ID, user.Username, auth.NormalizeEmail(user.Email), user.CreateTime.UTC())
	require.NoError(t, err)
	return user.ID
}

func newRegistrationUser(email string) types.User {
	id := uuid.New()
	return types.User{
		ID: id, Username: "registration-" + id.String(), Nickname: "Registration",
		Firstname: "Test", Lastname: "User", Email: email, PasswordHashed: "hash",
		HasLocalPassword: true, CreateTime: time.Now().UTC(), IsActive: true, Role: "user",
	}
}
