package group_test

import (
	"database/sql"
	"expense-tracker/config"
	"expense-tracker/db"
	"expense-tracker/services/auth"
	"expense-tracker/services/group"
	"expense-tracker/types"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestCreateGroup(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	store := group.NewStore(db)

	// define test cases
	type testcase struct {
		name        string
		mockGroup   types.Group
		expectFail  bool
		expectError error
	}

	subtests := []testcase{
		{
			name: "valid",
			mockGroup: types.Group{
				ID:           uuid.New(),
				GroupName:    "test",
				Description:  "test desc",
				CreateTime:   time.Now(),
				IsActive:     true,
				CreateByUser: uuid.New(),
			},
			expectFail:  false,
			expectError: nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			err := store.CreateGroup(test.mockGroup)
			defer deleteGroup(db, test.mockGroup.ID)
			defer deleteGroupMember(db, test.mockGroup.ID, []uuid.UUID{test.mockGroup.CreateByUser})

			assert.Equal(t, test.expectError, err)
		})
	}
}

func TestGetGroupByID(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	mockGroupID := uuid.New()
	mockGroup := types.Group{
		ID:           mockGroupID,
		GroupName:    "test group",
		Description:  "test desc",
		CreateTime:   time.Now(),
		IsActive:     true,
		CreateByUser: uuid.New(),
	}
	insertGroup(db, mockGroup)
	defer deleteGroup(db, mockGroupID)

	// define test cases
	type testcase struct {
		name        string
		mockID      string
		expectFail  bool
		expectError error
	}
	subtests := []testcase{
		{
			name:        "valid",
			mockID:      mockGroupID.String(),
			expectFail:  false,
			expectError: nil,
		},
		{
			name:        "invalid id",
			mockID:      uuid.NewString(),
			expectFail:  true,
			expectError: types.ErrGroupNotExist,
		},
	}

	store := group.NewStore(db)
	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			group, err := store.GetGroupByID(test.mockID)

			if test.expectFail {
				assert.Nil(t, group)
				assert.Equal(t, test.expectError, err)
			} else {
				assert.NotNil(t, group)
				assert.Equal(t, test.mockID, group.ID.String())
				assert.Nil(t, err)
			}
		})
	}
}

func TestGetGroupByIDAndUser(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	mockGroupID := uuid.New()
	mockGroup := types.Group{
		ID:           mockGroupID,
		GroupName:    "test group",
		Description:  "test desc",
		CreateTime:   time.Now(),
		IsActive:     true,
		CreateByUser: uuid.New(),
	}
	mockPassword, _ := auth.HashPassword("pword")
	mockUSerID := uuid.New()
	mockUser := types.User{
		ID:             mockUSerID,
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
	mockNonMemberUserID := uuid.New()
	mockNonMemberUser := types.User{
		ID:        mockNonMemberUserID,
		Username:  "testnonmember",
		Firstname: "fname",
		Lastname:  "lname",
	}
	insertUser(db, mockUser)
	insertUser(db, mockNonMemberUser)
	insertGroup(db, mockGroup)
	insertGroupMember(db, mockGroupID, []uuid.UUID{mockUSerID})
	defer cleanUser(db, mockUSerID)
	defer cleanUser(db, mockNonMemberUserID)
	defer deleteGroup(db, mockGroupID)
	defer deleteGroupMember(db, mockGroupID, []uuid.UUID{mockUSerID})

	// define test cases
	type testcase struct {
		name        string
		mockGroupID string
		mockUserID  string
		expectFail  bool
		expectGroup *types.Group
		expectError error
	}

	store := group.NewStore(db)
	subtests := []testcase{
		{
			name:        "valid",
			mockGroupID: mockGroupID.String(),
			mockUserID:  mockUSerID.String(),
			expectFail:  false,
			expectGroup: &types.Group{
				ID: mockGroupID,
			},
			expectError: nil,
		},
		{
			name:        "invalid group id",
			mockGroupID: uuid.NewString(),
			mockUserID:  mockUSerID.String(),
			expectFail:  true,
			expectGroup: nil,
			expectError: types.ErrGroupNotExist,
		},
		{
			name:        "invalid user id",
			mockGroupID: mockGroupID.String(),
			mockUserID:  uuid.NewString(),
			expectFail:  true,
			expectGroup: nil,
			expectError: types.ErrUserNotExist,
		},
		{
			name:        "invalid non-member user",
			mockGroupID: mockGroupID.String(),
			mockUserID:  mockNonMemberUserID.String(),
			expectFail:  true,
			expectGroup: nil,
			expectError: types.ErrUserNotPermitted,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			group, err := store.GetGroupByIDAndUser(test.mockGroupID, test.mockUserID)

			if test.expectFail {
				assert.Nil(t, group)
				assert.Equal(t, test.expectError, err)
			} else {
				assert.Equal(t, mockGroupID, group.ID)
				assert.Nil(t, err)
			}
		})
	}
}

func TestGetGroupListByUser(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	mockGroupID := uuid.New()
	mockGroup := types.Group{
		ID: mockGroupID,
	}
	mockGroupID2 := uuid.New()
	mockGroup2 := types.Group{
		ID: mockGroupID2,
	}
	mockUserID := uuid.New()
	mockUser := types.User{
		ID: mockUserID,
	}

	insertUser(db, mockUser)
	insertGroup(db, mockGroup)
	insertGroup(db, mockGroup2)
	insertGroupMember(db, mockGroupID, []uuid.UUID{mockUserID})
	insertGroupMember(db, mockGroupID2, []uuid.UUID{mockUserID})
	defer cleanUser(db, mockUserID)
	defer deleteGroup(db, mockGroupID)
	defer deleteGroup(db, mockGroupID2)
	defer deleteGroupMember(db, mockGroupID, []uuid.UUID{mockUserID})
	defer deleteGroupMember(db, mockGroupID2, []uuid.UUID{mockUserID})

	// define test cases
	type testcase struct {
		name         string
		mockUserID   string
		expectFail   bool
		expectGroups []*types.Group
		expectError  error
	}

	subtests := []testcase{
		{
			name:       "valid",
			mockUserID: mockUserID.String(),
			expectFail: false,
			expectGroups: []*types.Group{
				{
					ID: mockGroupID,
				},
				{
					ID: mockGroupID2,
				},
			},
			expectError: nil,
		},
		{
			name:         "invalid userid",
			mockUserID:   uuid.NewString(),
			expectFail:   true,
			expectGroups: nil,
			expectError:  nil,
		},
	}

	store := group.NewStore(db)
	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			groups, err := store.GetGroupListByUser(test.mockUserID)

			if test.expectFail {
				assert.Nil(t, groups)
				assert.Equal(t, test.expectError, err)
			} else {
				assert.NotNil(t, groups)
				assert.Equal(t, len(test.expectGroups), len(groups))
				assert.Nil(t, err)
			}
		})
	}
}

func TestGetGroupMemberByGroupID(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	mockGroupID := uuid.New()
	mockGroup := types.Group{
		ID: mockGroupID,
	}
	mockUserID := uuid.New()
	mockUser := types.User{
		ID: mockUserID,
	}
	mockUserID2 := uuid.New()
	mockUser2 := types.User{
		ID: mockUserID2,
	}

	insertUser(db, mockUser)
	insertUser(db, mockUser2)
	insertGroup(db, mockGroup)
	insertGroupMember(db, mockGroupID, []uuid.UUID{mockUserID})
	insertGroupMember(db, mockGroupID, []uuid.UUID{mockUserID2})
	defer cleanUser(db, mockUserID)
	defer cleanUser(db, mockUserID2)
	defer deleteGroup(db, mockGroupID)
	defer deleteGroupMember(db, mockGroupID, []uuid.UUID{mockUserID})
	defer deleteGroupMember(db, mockGroupID, []uuid.UUID{mockUserID2})

	// define test cases
	type testcase struct {
		name        string
		mockGroupID string
		expectFail  bool
		expectUsers []*types.User
		expectError error
	}

	subtests := []testcase{
		{
			name:        "valid",
			mockGroupID: mockGroupID.String(),
			expectFail:  false,
			expectUsers: []*types.User{
				{
					ID: mockUserID,
				},
				{
					ID: mockUserID2,
				},
			},
			expectError: nil,
		},
		{
			name:        "invalid group id",
			mockGroupID: uuid.NewString(),
			expectFail:  true,
			expectUsers: nil,
			expectError: nil,
		},
	}

	store := group.NewStore(db)
	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			users, err := store.GetGroupMemberByGroupID(test.mockGroupID)

			if test.expectFail {
				assert.Nil(t, users)
				assert.Equal(t, test.expectError, err)
			} else {
				assert.NotNil(t, users)
				assert.Equal(t, len(test.expectUsers), len(users))
				assert.Nil(t, err)
			}
		})
	}
}

func insertGroup(db *sql.DB, group types.Group) {
	createTime := group.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	statement := fmt.Sprintf(
		"INSERT INTO groups ("+
			"id, group_name, description, "+
			"create_time_utc, is_active, create_by_user_id"+
			") VALUES ('%s', '%s', '%s', '%s', '%t', '%s');",
		group.ID, group.GroupName, group.Description,
		createTime, group.IsActive, group.CreateByUser,
	)

	db.Exec(statement)
}

func deleteGroup(db *sql.DB, groupId uuid.UUID) {
	statement := fmt.Sprintf("DELETE FROM groups WHERE id='%s';", groupId)
	db.Exec(statement)
}

func insertGroupMember(db *sql.DB, groupId uuid.UUID, userids []uuid.UUID) {
	for _, id := range userids {
		statement := fmt.Sprintf(
			"INSERT INTO group_member ("+
				"id, group_id, user_id"+
				") VALUES ('%s', '%s', '%s');",
			uuid.NewString(), groupId, id,
		)
		db.Exec(statement)
	}
}

func deleteGroupMember(db *sql.DB, groupId uuid.UUID, userids []uuid.UUID) {
	for _, userid := range userids {
		statement := fmt.Sprintf(
			"DELETE FROM group_member "+
				"WHERE group_id='%s' AND user_id='%s';",
			groupId, userid,
		)
		db.Exec(statement)
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
