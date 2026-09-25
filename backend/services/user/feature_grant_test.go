package user_test

import (
	"expense-tracker/backend/services/user"
	"expense-tracker/backend/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestReceiptOCRGrantLifecycleAndAuthorization(t *testing.T) {
	database := openTestDB(t)
	adminID := uuid.New()
	regularID := uuid.New()
	targetID := uuid.New()
	inactiveID := uuid.New()
	for _, fixture := range []struct {
		id     uuid.UUID
		name   string
		active bool
	}{
		{id: adminID, name: "grant-admin", active: true},
		{id: regularID, name: "grant-regular", active: true},
		{id: targetID, name: "grant-target", active: true},
		{id: inactiveID, name: "grant-inactive", active: false},
	} {
		insertUser(database, types.User{
			ID: fixture.id, Username: fixture.name + "-" + fixture.id.String()[:8],
			Firstname: "Grant", Lastname: "Test",
			Email:          fixture.name + "-" + fixture.id.String()[:8] + "@example.test",
			PasswordHashed: "not-used", HasLocalPassword: true,
			CreateTime: time.Now(), IsActive: fixture.active,
		})
		t.Cleanup(func() { cleanUser(database, fixture.id) })
	}
	_, err := database.ExecContext(t.Context(), "UPDATE users SET role = 'admin' WHERE id = $1", adminID)
	require.NoError(t, err)

	store := user.NewStore(database)
	require.NoError(t, store.SetReceiptOCRGrant(t.Context(), adminID.String(), targetID.String(), true))
	granted, err := store.HasReceiptOCRGrant(t.Context(), targetID.String())
	require.NoError(t, err)
	require.True(t, granted)

	adminUsers, err := store.GetAdminUsers()
	require.NoError(t, err)
	var targetCapabilities types.UserCapabilities
	for _, account := range adminUsers {
		if account.ID == targetID {
			targetCapabilities = account.Capabilities
			break
		}
	}
	require.True(t, targetCapabilities.ReceiptOCR)

	require.ErrorIs(t,
		store.SetReceiptOCRGrant(t.Context(), regularID.String(), targetID.String(), false),
		types.ErrPermissionDenied,
	)
	require.ErrorIs(t,
		store.SetReceiptOCRGrant(t.Context(), adminID.String(), inactiveID.String(), true),
		types.ErrAccountInactive,
	)
	require.ErrorIs(t,
		store.SetReceiptOCRGrant(t.Context(), adminID.String(), uuid.NewString(), true),
		types.ErrUserNotExist,
	)

	_, err = database.ExecContext(t.Context(), "UPDATE users SET is_active = FALSE WHERE id = $1", targetID)
	require.NoError(t, err)
	granted, err = store.HasReceiptOCRGrant(t.Context(), targetID.String())
	require.NoError(t, err)
	require.False(t, granted, "inactive accounts must not have an effective grant")

	require.NoError(t, store.SetReceiptOCRGrant(t.Context(), adminID.String(), targetID.String(), false))
	_, err = database.ExecContext(t.Context(), "UPDATE users SET is_active = TRUE WHERE id = $1", targetID)
	require.NoError(t, err)
	granted, err = store.HasReceiptOCRGrant(t.Context(), targetID.String())
	require.NoError(t, err)
	require.False(t, granted)
}

func TestHasReceiptOCRGrantRejectsUnknownUser(t *testing.T) {
	database := openTestDB(t)
	granted, err := user.NewStore(database).HasReceiptOCRGrant(t.Context(), uuid.NewString())
	require.False(t, granted)
	require.ErrorIs(t, err, types.ErrUserNotExist)
}
