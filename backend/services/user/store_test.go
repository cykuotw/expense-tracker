package user_test

import (
	"database/sql"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/services/user"
	"expense-tracker/backend/types"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetUserByEmail(t *testing.T) {
	// prepare test data
	db := openTestDB(t)

	mockPassword, _ := auth.HashPassword("pword")
	mockEmail := "a@test.com"
	mockUser := types.User{
		ID:               uuid.New(),
		Username:         "testnickname",
		Nickname:         "testnickname",
		Firstname:        "testfirstname",
		Lastname:         "testlastname",
		Email:            mockEmail,
		PasswordHashed:   mockPassword,
		HasLocalPassword: true,
		ExternalType:     "",
		ExternalID:       "",
		CreateTime:       time.Now(),
		IsActive:         true,
	}
	insertUser(db, mockUser)
	defer cleanUser(db, mockUser.ID)

	// define test cases
	type testcase struct {
		name         string
		mockEmail    string
		expectFail   bool
		expectResult *types.User
		expectError  error
	}

	subtests := []testcase{
		{
			name:         "valid",
			mockEmail:    mockEmail,
			expectFail:   false,
			expectResult: &mockUser,
			expectError:  nil,
		},
		{
			name:         "invalid email",
			mockEmail:    "invalid@test.com",
			expectFail:   true,
			expectResult: nil,
			expectError:  types.ErrUserNotExist,
		},
	}
	store := user.NewStore(db)
	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			user, err := store.GetUserByEmail(test.mockEmail)

			if test.expectFail {
				assert.Equal(t, test.expectError, err)
			} else {
				assert.Equal(t, test.expectResult.ID, user.ID)
				assert.Nil(t, err)
			}
		})
	}
}

func TestGetUserByExternalIdentity(t *testing.T) {
	db := openTestDB(t)

	mockPassword, _ := auth.HashPassword("pword")
	mockUser := types.User{
		ID:             uuid.New(),
		Username:       "google-user",
		Nickname:       "google-user",
		Firstname:      "Google",
		Lastname:       "User",
		Email:          "google-user@example.com",
		PasswordHashed: mockPassword,
		ExternalType:   "google",
		ExternalID:     "google-sub-123",
		CreateTime:     time.Now(),
		IsActive:       true,
	}
	insertUser(db, mockUser)
	defer cleanUser(db, mockUser.ID)

	type testcase struct {
		name         string
		externalType string
		externalID   string
		expectFail   bool
		expectError  error
	}

	subtests := []testcase{
		{
			name:         "valid",
			externalType: "google",
			externalID:   "google-sub-123",
		},
		{
			name:         "invalid external identity",
			externalType: "google",
			externalID:   "missing-sub",
			expectFail:   true,
			expectError:  types.ErrUserNotExist,
		},
	}

	store := user.NewStore(db)
	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			user, err := store.GetUserByExternalIdentity(test.externalType, test.externalID)

			if test.expectFail {
				assert.Equal(t, test.expectError, err)
				return
			}

			assert.NoError(t, err)
			assert.Equal(t, mockUser.ID, user.ID)
			assert.Equal(t, mockUser.ExternalID, user.ExternalID)
		})
	}
}

func TestGetUserByID(t *testing.T) {
	// prepare test data
	db := openTestDB(t)

	mockPassword, _ := auth.HashPassword("pword")
	mockID := uuid.New()
	mockUser := types.User{
		ID:               mockID,
		Username:         "testnickname",
		Nickname:         "testnickname",
		Firstname:        "testfirstname",
		Lastname:         "testlastname",
		Email:            "a@test.com",
		PasswordHashed:   mockPassword,
		HasLocalPassword: true,
		ExternalType:     "",
		ExternalID:       "",
		CreateTime:       time.Now(),
		IsActive:         true,
	}
	insertUser(db, mockUser)
	defer cleanUser(db, mockUser.ID)

	// define test cases
	type testcase struct {
		name         string
		mockID       uuid.UUID
		expectFail   bool
		expectResult *types.User
		expectError  error
	}

	subtests := []testcase{
		{
			name:         "valid",
			mockID:       mockID,
			expectFail:   false,
			expectResult: &mockUser,
			expectError:  nil,
		},
		{
			name:         "invalid user",
			mockID:       uuid.New(),
			expectFail:   true,
			expectResult: nil,
			expectError:  types.ErrUserNotExist,
		},
	}
	store := user.NewStore(db)
	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			user, err := store.GetUserByID(test.mockID.String())

			if test.expectFail {
				assert.Equal(t, test.expectError, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.expectResult.ID, user.ID)
			}
		})
	}
}

func TestGetUsernameByID(t *testing.T) {
	// prepare test data
	db := openTestDB(t)

	mockPassword, _ := auth.HashPassword("pword")
	mockID := uuid.New()
	mockUsername := "testusername"
	mockUser := types.User{
		ID:               mockID,
		Username:         mockUsername,
		Nickname:         mockUsername,
		Firstname:        "testfirstname",
		Lastname:         "testlastname",
		Email:            "a@test.com",
		PasswordHashed:   mockPassword,
		HasLocalPassword: true,
		ExternalType:     "",
		ExternalID:       "",
		CreateTime:       time.Now(),
		IsActive:         true,
	}
	insertUser(db, mockUser)
	defer cleanUser(db, mockUser.ID)

	// define test cases
	type testcase struct {
		name         string
		mockID       uuid.UUID
		expectFail   bool
		expectResult string
		expectError  error
	}

	subtests := []testcase{
		{
			name:         "valid",
			mockID:       mockID,
			expectFail:   false,
			expectResult: mockUsername,
			expectError:  nil,
		},
		{
			name:         "invalid userid",
			mockID:       uuid.New(),
			expectFail:   true,
			expectResult: "",
			expectError:  types.ErrUserNotExist,
		},
	}
	store := user.NewStore(db)
	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			username, err := store.GetUsernameByID(test.mockID.String())

			if test.expectFail {
				assert.Equal(t, test.expectError, err)
				assert.Zero(t, len(username))
			} else {
				assert.Equal(t, test.expectResult, username)
				assert.Nil(t, err)
			}
		})
	}
}

func TestGetUsernamesByIDsUsesCurrentProfileDisplayName(t *testing.T) {
	db := openTestDB(t)
	userID := uuid.New()
	profileUser := types.User{
		ID:               userID,
		Username:         "ledger-account",
		Nickname:         "Ledger name",
		Firstname:        "Taylor",
		Lastname:         "Swift",
		Email:            "ledger-name@example.com",
		PasswordHashed:   "hash",
		HasLocalPassword: true,
		CreateTime:       time.Now(),
		IsActive:         true,
	}
	insertUser(db, profileUser)
	defer cleanUser(db, userID)

	store := user.NewStore(db)
	names, err := store.GetUsernamesByIDs([]string{userID.String()})
	assert.NoError(t, err)
	assert.Equal(t, "Ledger name", names[userID.String()])

	_, err = db.Exec("UPDATE users SET nickname = $1 WHERE id = $2", "", userID)
	assert.NoError(t, err)
	names, err = store.GetUsernamesByIDs([]string{userID.String()})
	assert.NoError(t, err)
	assert.Equal(t, "Taylor Swift", names[userID.String()])
}

func insertUser(db *sql.DB, user types.User) {
	createTime := user.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	query := fmt.Sprintf(
		"INSERT INTO users ("+
			"id, username, firstname, lastname, nickname, "+
			"email, password_hash, "+
			"has_local_password, "+
			"external_type, external_id, "+
			"create_time_utc, is_active"+
			") VALUES ('%s','%s','%s','%s','%s','%s','%s',%t,%s,%s,'%s',%t);",
		user.ID, user.Username, user.Firstname, user.Lastname, user.Nickname,
		user.Email, user.PasswordHashed, user.HasLocalPassword,
		sqlStringOrNull(user.ExternalType), sqlStringOrNull(user.ExternalID),
		createTime, user.IsActive,
	)
	db.Exec(query)
}

func sqlStringOrNull(value string) string {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "NULL"
	}
	return fmt.Sprintf("'%s'", normalized)
}

func cleanUser(db *sql.DB, id uuid.UUID) {
	query := fmt.Sprintf("DELETE FROM users WHERE id = '%s'", id)
	db.Exec(query)
}
