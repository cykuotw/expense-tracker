package store

func (s *Store) CheckGroupBalanceAllSettled(groupID string) (bool, error) {
	query := "SELECT NOT EXISTS (SELECT 1 FROM balance WHERE group_id = $1 AND is_outdated = FALSE AND is_settled = FALSE);"
	rows, err := s.db.Query(query, groupID)
	if err != nil {
		return false, err
	}
	defer rows.Close()

	notExist := false
	for rows.Next() {
		err := rows.Scan(&notExist)
		if err != nil {
			return false, err
		}
	}
	if err := rows.Err(); err != nil {
		return false, err
	}

	return notExist, nil
}
