package group_test

import (
	"database/sql"
	"expense-tracker/config"
	"expense-tracker/db"
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

func insertGroupMember(db *sql.DB, groupId uuid.UUID, members []types.User) {

}

func deleteGroupMember(db *sql.DB, groupId uuid.UUID, userids []uuid.UUID) {

}
