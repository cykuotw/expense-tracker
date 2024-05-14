package group

import (
	"database/sql"
	"expense-tracker/types"
	"fmt"
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
	return nil, nil
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
