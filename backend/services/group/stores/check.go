package group

import (
	"database/sql"
	"errors"
)

func (s *Store) CheckGroupExistById(id string) (bool, error) {
	query := "SELECT EXISTS (SELECT 1 FROM groups WHERE id = $1);"
	exist := false
	if err := s.db.QueryRow(query, id).Scan(&exist); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return exist, nil
}

func (s *Store) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	query := "SELECT EXISTS (SELECT 1 FROM group_member WHERE group_id = $1 AND user_id = $2);"
	exist := false
	if err := s.db.QueryRow(query, groupId, userId).Scan(&exist); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return false, err
	}
	return exist, nil
}
