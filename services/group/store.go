package group

import (
	"database/sql"
	"expense-tracker/types"
	"fmt"

	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateGroup(group types.Group) error {
	createTime := group.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	statement := fmt.Sprintf(
		"INSERT INTO groups ("+
			"id, group_name, description, "+
			"create_time_utc, is_active, create_by_user_id"+
			") VALUES ('%s', '%s', '%s', '%s', '%t', '%s');",
		group.ID, group.GroupName, group.Description,
		createTime, group.IsActive, group.CreateByUser,
	)

	_, err := s.db.Exec(statement)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) GetGroupByID(id string) (*types.Group, error) {
	statement := fmt.Sprintf("SELECT * FROM groups WHERE id='%s';", id)
	rows, err := s.db.Query(statement)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	group := new(types.Group)
	for rows.Next() {
		group, err = scanRowIntoGroup(rows)
		if err != nil {
			return nil, err
		}
	}

	if group.ID == uuid.Nil {
		return nil, types.ErrGroupNotExist
	}

	return group, nil
}

func (s *Store) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	return nil, nil
}

func (s *Store) GetGroupListByUser(userID string) ([]*types.Group, error) {
	return nil, nil
}

func (s *Store) GetGroupMemberByGroupID(groupID string) ([]*types.User, error) {
	return nil, nil
}

func (s *Store) UpdateGroupMember(action string, userID string, groupID string) error {
	return nil
}

func (s *Store) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}

func scanRowIntoGroup(rows *sql.Rows) (*types.Group, error) {
	group := new(types.Group)

	err := rows.Scan(
		&group.ID,
		&group.GroupName,
		&group.Description,
		&group.CreateTime,
		&group.IsActive,
		&group.CreateByUser,
	)
	if err != nil {
		return nil, err
	}
	return group, nil
}
