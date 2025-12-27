package group

func (s *Store) CheckGroupExistById(id string) (bool, error) {
	query := "SELECT EXISTS (SELECT 1 FROM groups WHERE id = $1);"
	rows, err := s.db.Query(query, id)
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
	query := "SELECT EXISTS (SELECT 1 FROM group_member WHERE group_id = $1 AND user_id = $2);"
	rows, err := s.db.Query(query, groupId, userId)
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
