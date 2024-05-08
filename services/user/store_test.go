package user_test

import (
	"database/sql"
	"expense-tracker/config"
	"expense-tracker/db"
	"expense-tracker/services/auth"
	"expense-tracker/services/user"
	"expense-tracker/types"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestGetUserByEmail(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)

	mockPassword, _ := auth.HashPassword("pword")
	mockEmail := "a@test.com"
	mockUser := types.User{
		ID:             uuid.New(),
		Username:       "testusername",
		Firstname:      "testfirstname",
		Lastname:       "testlastname",
		Email:          mockEmail,
		PasswordHashed: mockPassword,
		ExternalType:   "",
		ExternalID:     "",
		CreateTime:     time.Now(),
		IsActive:       true,
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

func TestGetUserByUsername(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)

	mockPassword, _ := auth.HashPassword("pword")
	mockUsername := "testusername"
	mockUser := types.User{
		ID:             uuid.New(),
		Username:       mockUsername,
		Firstname:      "testfirstname",
		Lastname:       "testlastname",
		Email:          "a@test.com",
		PasswordHashed: mockPassword,
		ExternalType:   "",
		ExternalID:     "",
		CreateTime:     time.Now(),
		IsActive:       true,
	}
	insertUser(db, mockUser)
	defer cleanUser(db, mockUser.ID)

	// define test cases
	type testcase struct {
		name         string
		mockUsername string
		expectFail   bool
		expectResult *types.User
		expectError  error
	}

	subtests := []testcase{
		{
			name:         "valid",
			mockUsername: mockUsername,
			expectFail:   false,
			expectResult: &mockUser,
			expectError:  nil,
		},
		{
			name:         "invalid user",
			mockUsername: "invalidusername",
			expectFail:   true,
			expectResult: nil,
			expectError:  types.ErrUserNotExist,
		},
	}
	store := user.NewStore(db)
	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			user, err := store.GetUserByUsername(test.mockUsername)

			if test.expectFail {
				assert.Equal(t, test.expectError, err)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.expectResult.ID, user.ID)
			}
		})
	}
}

func TestGetUserByID(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)

	mockPassword, _ := auth.HashPassword("pword")
	mockID := uuid.New()
	mockUser := types.User{
		ID:             mockID,
		Username:       "testusername",
		Firstname:      "testfirstname",
		Lastname:       "testlastname",
		Email:          "a@test.com",
		PasswordHashed: mockPassword,
		ExternalType:   "",
		ExternalID:     "",
		CreateTime:     time.Now(),
		IsActive:       true,
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

func insertUser(db *sql.DB, user types.User) {
	createTime := user.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	statement := fmt.Sprintf(
		"INSERT INTO users ("+
			"id, username, firstname, lastname, "+
			"email, password_hash, "+
			"external_type, external_id, "+
			"create_time_utc, is_active"+
			") VALUES ('%s','%s','%s','%s','%s','%s','%s','%s','%s',%t);",
		user.ID, user.Username, user.Firstname, user.Lastname,
		user.Email, user.PasswordHashed,
		user.ExternalType, user.ExternalID,
		createTime, user.IsActive,
	)
	db.Exec(statement)
}

func cleanUser(db *sql.DB, id uuid.UUID) {
	statement := fmt.Sprintf("DELETE FROM users WHERE id = '%s'", id)
	db.Exec(statement)
}
