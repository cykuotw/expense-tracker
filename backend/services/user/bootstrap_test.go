package user

import (
	"errors"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type mockFirstAdminStore struct {
	reconcileFn func(*types.User, string) (BootstrapStatus, error)
}

func (m *mockFirstAdminStore) ReconcileFirstAdmin(candidate *types.User, normalizedEmail string) (BootstrapStatus, error) {
	return m.reconcileFn(candidate, normalizedEmail)
}

func TestBootstrapFirstAdminCreatesSeededAdminCandidate(t *testing.T) {
	expectedID := uuid.New()
	expectedTime := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)
	store := &mockFirstAdminStore{reconcileFn: func(candidate *types.User, email string) (BootstrapStatus, error) {
		assert.NotNil(t, candidate)
		assert.Equal(t, expectedID, candidate.ID)
		assert.Equal(t, "admin@example.com", email)
		assert.Equal(t, "admin@example.com", candidate.Email)
		assert.Equal(t, "Ada", candidate.Firstname)
		assert.Equal(t, "Lovelace", candidate.Lastname)
		assert.Equal(t, "Ada Lovelace", candidate.Username)
		assert.Equal(t, "", candidate.Nickname)
		assert.Equal(t, "admin", candidate.Role)
		assert.True(t, candidate.IsActive)
		assert.True(t, candidate.HasLocalPassword)
		assert.Equal(t, expectedTime, candidate.CreateTime)
		assert.True(t, auth.ValidatePassword(candidate.PasswordHashed, "supersecret"))
		return BootstrapStatusCreated, nil
	}}
	input := &FirstAdminInput{
		Email: " ADMIN@example.com ", Password: "supersecret",
		Firstname: " Ada ", Lastname: " Lovelace ",
	}

	status, err := BootstrapFirstAdmin(store, input, BootstrapDeps{
		Now: func() time.Time { return expectedTime }, NewUUID: func() uuid.UUID { return expectedID },
	})

	assert.NoError(t, err)
	assert.Equal(t, BootstrapStatusCreated, status)
}

func TestBootstrapFirstAdminUsesNicknameAsUsername(t *testing.T) {
	store := &mockFirstAdminStore{reconcileFn: func(candidate *types.User, _ string) (BootstrapStatus, error) {
		assert.Equal(t, "ada", candidate.Username)
		assert.Equal(t, "ada", candidate.Nickname)
		return BootstrapStatusCreated, nil
	}}
	input := &FirstAdminInput{
		Email: "admin@example.com", Password: "supersecret",
		Firstname: "Ada", Lastname: "Lovelace", Nickname: "ada",
	}

	status, err := BootstrapFirstAdmin(store, input, BootstrapDeps{})

	assert.NoError(t, err)
	assert.Equal(t, BootstrapStatusCreated, status)
}

func TestBootstrapFirstAdminWithoutConfigurationDelegatesReconciliation(t *testing.T) {
	store := &mockFirstAdminStore{reconcileFn: func(candidate *types.User, email string) (BootstrapStatus, error) {
		assert.Nil(t, candidate)
		assert.Empty(t, email)
		return BootstrapStatusNotRequested, nil
	}}

	status, err := BootstrapFirstAdmin(store, nil, BootstrapDeps{})

	assert.NoError(t, err)
	assert.Equal(t, BootstrapStatusNotRequested, status)
}

func TestBootstrapFirstAdminRejectsInvalidInput(t *testing.T) {
	store := &mockFirstAdminStore{reconcileFn: func(*types.User, string) (BootstrapStatus, error) {
		t.Fatal("store must not be called for invalid input")
		return "", nil
	}}
	input := &FirstAdminInput{
		Email: "invalid-email", Password: "short",
		Firstname: "Ada", Lastname: "Lovelace",
	}

	status, err := BootstrapFirstAdmin(store, input, BootstrapDeps{})

	assert.Error(t, err)
	assert.Empty(t, status)
}

func TestBootstrapFirstAdminPropagatesStoreErrors(t *testing.T) {
	expectedErr := errors.New("db down")
	store := &mockFirstAdminStore{reconcileFn: func(*types.User, string) (BootstrapStatus, error) {
		return "", expectedErr
	}}
	input := &FirstAdminInput{
		Email: "admin@example.com", Password: "supersecret",
		Firstname: "Ada", Lastname: "Lovelace",
	}

	status, err := BootstrapFirstAdmin(store, input, BootstrapDeps{})

	assert.ErrorIs(t, err, expectedErr)
	assert.Empty(t, status)
}
