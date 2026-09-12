package notification

import (
	"context"
	"database/sql"
	"expense-tracker/backend/config"
	"expense-tracker/backend/db"
	"expense-tracker/backend/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSubscriptionSettingsAndGroupMute(t *testing.T) {
	database := openTestDB(t)
	defer database.Close()
	ctx := t.Context()
	userID := uuid.New()
	groupID := uuid.New()
	require.NoError(t, insertUser(ctx, database, userID))
	require.NoError(t, insertGroupMember(ctx, database, groupID, userID))
	defer cleanupNotificationFixtures(t, database, userID, groupID)

	store := NewStore(database)
	input := types.WebPushSubscriptionInput{
		Endpoint: "https://fcm.googleapis.com/fcm/send/test-" + uuid.NewString(),
		P256DH:   "B" + "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		Auth:     "AAAAAAAAAAAAAAAAAAAAAA",
	}
	require.NoError(t, store.UpsertSubscription(ctx, userID, input))
	require.NoError(t, store.SetDetails(ctx, userID, input.Endpoint, true))
	require.NoError(t, store.SetGroupMute(ctx, userID, groupID, true))

	settings, err := store.Settings(ctx, userID, "public-key")
	require.NoError(t, err)
	assert.True(t, settings.Enabled)
	assert.True(t, settings.ShowDetails)
	assert.Equal(t, "public-key", settings.VAPIDKey)
	require.Len(t, settings.MutedGroups, 1)
	assert.Equal(t, groupID, settings.MutedGroups[0].GroupID)
	status, err := store.SubscriptionStatus(ctx, userID, input.Endpoint)
	require.NoError(t, err)
	assert.True(t, status.Registered)
	assert.True(t, status.ShowDetails)

	status, err = store.SubscriptionStatus(ctx, userID, "https://fcm.googleapis.com/fcm/send/missing")
	require.NoError(t, err)
	assert.False(t, status.Registered)
	assert.False(t, status.ShowDetails)

	require.NoError(t, store.UpsertSubscription(ctx, userID, input))
	settings, err = store.Settings(ctx, userID, "public-key")
	require.NoError(t, err)
	assert.False(t, settings.ShowDetails, "replacing a subscription resets previews")
}

func TestValidateSubscriptionRejectsUnsafeEndpoint(t *testing.T) {
	ctx := t.Context()
	input := types.WebPushSubscriptionInput{
		Endpoint: "https://127.0.0.1/push",
		P256DH:   "B" + "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA",
		Auth:     "AAAAAAAAAAAAAAAAAAAAAA",
	}
	assert.ErrorIs(t, validateSubscription(ctx, input), types.ErrInvalidWebPushSubscription)
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	database, err := db.NewPostgreSQLStorage(config.Envs)
	require.NoError(t, err)
	require.NoError(t, database.Ping())
	return database
}

func insertUser(ctx context.Context, database *sql.DB, userID uuid.UUID) error {
	_, err := database.ExecContext(ctx, `
		INSERT INTO users (id, username, firstname, lastname, email, password_hash, create_time_utc, is_active, has_local_password, role)
		VALUES ($1, $2, 'Test', 'User', $3, 'unused', $4, TRUE, TRUE, 'user')`,
		userID, "notification-"+userID.String()[:8], userID.String()[:8]+"@example.test", time.Now().UTC())
	return err
}

func insertGroupMember(ctx context.Context, database *sql.DB, groupID, userID uuid.UUID) error {
	if _, err := database.ExecContext(ctx, `
		INSERT INTO groups (id, group_name, description, create_time_utc, is_active, create_by_user_id, currency)
		VALUES ($1, $2, '', $3, TRUE, $4, 'CAD')`,
		groupID, "notification-"+groupID.String()[:8], time.Now().UTC(), userID); err != nil {
		return err
	}
	_, err := database.ExecContext(ctx, "INSERT INTO group_member (id, group_id, user_id) VALUES ($1, $2, $3)", uuid.New(), groupID, userID)
	return err
}

func cleanupNotificationFixtures(t *testing.T, database *sql.DB, userID, groupID uuid.UUID) {
	t.Helper()
	_, err := database.Exec("DELETE FROM groups WHERE id = $1", groupID)
	require.NoError(t, err)
	_, err = database.Exec("DELETE FROM users WHERE id = $1", userID)
	require.NoError(t, err)
}
