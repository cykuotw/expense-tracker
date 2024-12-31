package group

import (
	"fmt"
)

func (s *Store) CheckGroupExistById(id string) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM groups WHERE id = '%s')", id)
	rows, err := s.db.Query(query)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	exist := false
	for rows.Next() {
		err := rows.Scan(&exist)
		if err != nil {
			return false, err
		}
	}

	return exist, nil
}

func (s *Store) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	query := fmt.Sprintf(`
			SELECT EXISTS (
				SELECT 1 FROM group_member WHERE group_id='%s' AND user_id='%s');`,
		groupId, userId,
	)
	rows, err := s.db.Query(query)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	exist := false
	for rows.Next() {
		if err := rows.Scan(&exist); err != nil {
			return false, err
		}
	}

	return exist, nil
}
