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
				Currency:     "CAD",
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
func TestGetGroupCurrency(t *testing.T) {
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	mockGroupID := uuid.New()
	mockCurrency := "CAD"
	mockGroup := types.Group{
		ID:       mockGroupID,
		Currency: mockCurrency,
	}
	insertGroup(db, mockGroup)
	defer deleteGroup(db, mockGroupID)

	// define test cases
	type testcase struct {
		name           string
		mockGroupID    uuid.UUID
		expectFail     bool
		expectCurrency string
		expectError    error
	}

	subtests := []testcase{
		{
			name:           "valid",
			mockGroupID:    mockGroupID,
			expectFail:     false,
			expectCurrency: mockCurrency,
			expectError:    nil,
		},
		{
			name:           "invalid group id",
			mockGroupID:    uuid.New(),
			expectFail:     true,
			expectCurrency: "",
			expectError:    types.ErrGroupNotExist,
		},
	}

	store := group.NewStore(db)
	for _, test := range subtests {
		currency, err := store.GetGroupCurrency(test.mockGroupID.String())

		if test.expectFail {
			assert.Zero(t, len(currency))
			assert.Equal(t, test.expectError, err)
		} else {
			assert.Equal(t, test.expectCurrency, currency)
			assert.Nil(t, err)
		}

	}
}

func TestUpdateGroupMember(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	mockGroupID := uuid.New()
	mockUserID := uuid.New()

	// define test cases
	type testcase struct {
		name        string
		action      string
		mockGroupID string
		mockUserID  string
		expectFail  bool
		expectError error
	}

	subtests := []testcase{
		{
			name:        "valid add",
			action:      "add",
			mockGroupID: mockGroupID.String(),
			mockUserID:  mockUserID.String(),
			expectFail:  false,
			expectError: nil,
		},
		{
			name:        "valid delete",
			action:      "delete",
			mockGroupID: mockGroupID.String(),
			mockUserID:  mockUserID.String(),
			expectFail:  false,
			expectError: nil,
		},
	}

	store := group.NewStore(db)
	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			if test.action == "delete" {
				insertGroupMember(db, uuid.MustParse(test.mockGroupID), []uuid.UUID{uuid.MustParse(test.mockUserID)})
			}
			defer deleteGroupMember(db, uuid.MustParse(test.mockGroupID), []uuid.UUID{uuid.MustParse(test.mockUserID)})

			err := store.UpdateGroupMember(test.action, test.mockUserID, test.mockGroupID)

			if test.expectFail {
				userid := getGroupMember(db, uuid.MustParse(test.mockGroupID), uuid.MustParse(test.mockUserID))
				if test.action == "add" {
					assert.Equal(t, test.mockUserID, userid)
					defer deleteGroupMember(db, uuid.MustParse(test.mockGroupID), []uuid.UUID{uuid.MustParse(test.mockUserID)})
				} else if test.action == "delete" {
					assert.Equal(t, uuid.Nil, userid)
				}
			} else {
				userid := getGroupMember(db, uuid.MustParse(test.mockGroupID), uuid.MustParse(test.mockUserID))
				if test.action == "add" {
					assert.Equal(t, test.mockUserID, userid.String())
				} else if test.action == "delete" {
					assert.Equal(t, uuid.Nil, userid)
				}
				assert.Nil(t, err)
			}
		})
	}
}

func TestUpdateGroupStatus(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	mockGroupIDT := uuid.New()
	mockGroupT := types.Group{
		ID:       mockGroupIDT,
		IsActive: true,
	}
	mockGroupIDF := uuid.New()
	mockGroupF := types.Group{
		ID:       mockGroupIDF,
		IsActive: true,
	}

	insertGroup(db, mockGroupT)
	insertGroup(db, mockGroupF)
	defer deleteGroup(db, mockGroupIDT)
	defer deleteGroup(db, mockGroupIDF)

	// define test cases
	type testcase struct {
		name        string
		isActive    bool
		groupID     string
		expectFail  bool
		expectError error
	}

	subtests := []testcase{
		{
			name:        "valid set to false",
			isActive:    false,
			groupID:     mockGroupIDT.String(),
			expectFail:  false,
			expectError: nil,
		},
		{
			name:        "valid set to true",
			isActive:    false,
			groupID:     mockGroupIDF.String(),
			expectFail:  false,
			expectError: nil,
		},
		{
			name:        "invalid group id",
			isActive:    false,
			groupID:     uuid.NewString(),
			expectFail:  true,
			expectError: nil,
		},
	}

	store := group.NewStore(db)
	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			err := store.UpdateGroupStatus(test.groupID, test.isActive)

			group := getGroup(db, uuid.MustParse(test.groupID))
			if test.expectFail {
				assert.Equal(t, group.ID, uuid.Nil)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.isActive, group.IsActive)
			}
		})
	}
}

func insertGroup(db *sql.DB, group types.Group) {
	createTime := group.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	query := fmt.Sprintf(
		"INSERT INTO groups ("+
			"id, group_name, description, "+
			"create_time_utc, is_active, create_by_user_id, currency"+
			") VALUES ('%s', '%s', '%s', '%s', '%t', '%s', '%s');",
		group.ID, group.GroupName, group.Description,
		createTime, group.IsActive, group.CreateByUser, group.Currency,
	)

	db.Exec(query)
}

func getGroup(db *sql.DB, groupId uuid.UUID) types.Group {
	query := fmt.Sprintf("SELECT * FROM groups WHERE id='%s';", groupId.String())
	rows, _ := db.Query(query)

	group := types.Group{}
	for rows.Next() {
		rows.Scan(
			&group.ID,
			&group.GroupName,
			&group.Description,
			&group.CreateTime,
			&group.IsActive,
			&group.CreateByUser,
		)
	}
	return group
}

func deleteGroup(db *sql.DB, groupId uuid.UUID) {
	query := fmt.Sprintf("DELETE FROM groups WHERE id='%s';", groupId)
	db.Exec(query)
}

func insertGroupMember(db *sql.DB, groupId uuid.UUID, userids []uuid.UUID) {
	for _, id := range userids {
		query := fmt.Sprintf(
			"INSERT INTO group_member ("+
				"id, group_id, user_id"+
				") VALUES ('%s', '%s', '%s');",
			uuid.NewString(), groupId, id,
		)
		db.Exec(query)
	}
}

func getGroupMember(db *sql.DB, groupId uuid.UUID, userid uuid.UUID) uuid.UUID {
	query := fmt.Sprintf(
		"SELECT user_id FROM group_member WHERE group_id='%s' AND user_id='%s';",
		groupId, userid,
	)

	rows, _ := db.Query(query)
	defer rows.Close()

	var id uuid.UUID
	for rows.Next() {
		rows.Scan(&id)
	}
	return id
}

func deleteGroupMember(db *sql.DB, groupId uuid.UUID, userids []uuid.UUID) {
	for _, userid := range userids {
		query := fmt.Sprintf(
			"DELETE FROM group_member "+
				"WHERE group_id='%s' AND user_id='%s';",
			groupId, userid,
		)
		db.Exec(query)
	}
}

func insertUser(db *sql.DB, user types.User) {
	createTime := user.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	query := fmt.Sprintf(
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
	db.Exec(query)
}

func cleanUser(db *sql.DB, id uuid.UUID) {
	query := fmt.Sprintf("DELETE FROM users WHERE id = '%s'", id)
	db.Exec(query)
}
