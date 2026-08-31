package invitation_test

import (
	"database/sql"
	"expense-tracker/backend/config"
	"expense-tracker/backend/db"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	cfg := config.Envs
	conn, err := db.NewPostgreSQLStorage(cfg)
	if err != nil {
		t.Skipf("skipping: db connect error: %v", err)
	}
	err = conn.Ping()
	if err != nil {
		t.Skipf("skipping: db ping error: %v", err)
	}
	return conn
}

func setupUser(t *testing.T, conn *sql.DB, id uuid.UUID) {
	t.Helper()
	_, err := conn.Exec(`INSERT INTO users (id, username, firstname, lastname, nickname, email, password_hash, create_time_utc, is_active, role)
		VALUES ($1, $2, 'Test', 'User', 'Test', $3, 'hash', $4, TRUE, 'admin')`,
		id, "invite-"+id.String()[:8], "invitation-"+id.String()+"@example.test", time.Now().UTC())
	require.NoError(t, err)
}

func cleanUser(t *testing.T, conn *sql.DB, id uuid.UUID) {
	t.Helper()
	_, err := conn.Exec("DELETE FROM users WHERE id = $1", id)
	require.NoError(t, err)
}

func cleanInvitation(t *testing.T, conn *sql.DB, id uuid.UUID) {
	t.Helper()
	_, err := conn.Exec("DELETE FROM invitations WHERE id = $1", id)
	require.NoError(t, err)
}
