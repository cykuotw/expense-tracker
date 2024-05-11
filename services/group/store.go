package group

import (
	"database/sql"
	"expense-tracker/types"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateGroup(group types.Group) error {
	return nil
}

func (s *Store) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}

func (s *Store) GetGroupListByUser(userID string) ([]*types.Group, error) {
	return nil, nil
}

func (s *Store) GetGroupMemberByGroupID(groupID string) ([]*types.User, error) {
	return nil, nil
}

func (s *Store) UpdateGroupMember(action string, userID string) error {
	return nil
}

func (s *Store) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
