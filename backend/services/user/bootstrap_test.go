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
	CheckAdminUserExistsFn func() (bool, error)
	CreateUserFn           func(types.User) error
}

func (m *mockFirstAdminStore) CheckAdminUserExists() (bool, error) {
	if m.CheckAdminUserExistsFn != nil {
		return m.CheckAdminUserExistsFn()
	}
	return false, nil
}

func (m *mockFirstAdminStore) CreateUser(user types.User) error {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(user)
	}
	return nil
}

func TestBootstrapFirstAdminCreatesSeededAdmin(t *testing.T) {
	expectedID := uuid.New()
	expectedTime := time.Date(2026, 3, 26, 12, 0, 0, 0, time.UTC)

	store := &mockFirstAdminStore{
		CheckAdminUserExistsFn: func() (bool, error) {
			return false, nil
		},
		CreateUserFn: func(user types.User) error {
			assert.Equal(t, expectedID, user.ID)
			assert.Equal(t, "admin@example.com", user.Email)
			assert.Equal(t, "Ada", user.Firstname)
			assert.Equal(t, "Lovelace", user.Lastname)
			assert.Equal(t, "Ada Lovelace", user.Username)
			assert.Equal(t, "", user.Nickname)
			assert.Equal(t, "admin", user.Role)
			assert.True(t, user.IsActive)
			assert.Equal(t, expectedTime, user.CreateTime)
			assert.True(t, auth.ValidatePassword(user.PasswordHashed, "supersecret"))
			return nil
		},
	}

	created, err := BootstrapFirstAdmin(store, FirstAdminInput{
		Email:     " admin@example.com ",
		Password:  "supersecret",
		Firstname: " Ada ",
		Lastname:  " Lovelace ",
	}, BootstrapDeps{
		Now:     func() time.Time { return expectedTime },
		NewUUID: func() uuid.UUID { return expectedID },
	})

	assert.NoError(t, err)
	assert.True(t, created)
}

func TestBootstrapFirstAdminUsesNicknameAsUsername(t *testing.T) {
	store := &mockFirstAdminStore{
		CheckAdminUserExistsFn: func() (bool, error) {
			return false, nil
		},
		CreateUserFn: func(user types.User) error {
			assert.Equal(t, "ada", user.Username)
			assert.Equal(t, "ada", user.Nickname)
			return nil
		},
	}

	created, err := BootstrapFirstAdmin(store, FirstAdminInput{
		Email:     "admin@example.com",
		Password:  "supersecret",
		Firstname: "Ada",
		Lastname:  "Lovelace",
		Nickname:  "ada",
	}, BootstrapDeps{})

	assert.NoError(t, err)
	assert.True(t, created)
}

func TestBootstrapFirstAdminSkipsWhenAdminAlreadyExists(t *testing.T) {
	store := &mockFirstAdminStore{
		CheckAdminUserExistsFn: func() (bool, error) {
			return true, nil
		},
		CreateUserFn: func(user types.User) error {
			t.Fatalf("CreateUser should not be called when an admin already exists")
			return nil
		},
	}

	created, err := BootstrapFirstAdmin(store, FirstAdminInput{}, BootstrapDeps{})

	assert.NoError(t, err)
	assert.False(t, created)
}

func TestBootstrapFirstAdminRejectsInvalidInput(t *testing.T) {
	store := &mockFirstAdminStore{
		CheckAdminUserExistsFn: func() (bool, error) {
			return false, nil
		},
	}

	created, err := BootstrapFirstAdmin(store, FirstAdminInput{
		Email:     "invalid-email",
		Password:  "short",
		Firstname: "Ada",
		Lastname:  "Lovelace",
	}, BootstrapDeps{})

	assert.Error(t, err)
	assert.False(t, created)
}

func TestBootstrapFirstAdminPropagatesStoreErrors(t *testing.T) {
	expectedErr := errors.New("db down")
	store := &mockFirstAdminStore{
		CheckAdminUserExistsFn: func() (bool, error) {
			return false, expectedErr
		},
	}

	created, err := BootstrapFirstAdmin(store, FirstAdminInput{
		Email:     "admin@example.com",
		Password:  "supersecret",
		Firstname: "Ada",
		Lastname:  "Lovelace",
	}, BootstrapDeps{})

	assert.ErrorIs(t, err, expectedErr)
	assert.False(t, created)
}
