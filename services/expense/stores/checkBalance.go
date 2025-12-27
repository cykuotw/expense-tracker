package store

func (s *Store) CheckBalanceExistByID(id string) (bool, error) {
	query := "SELECT EXISTS (SELECT 1 FROM balance WHERE id = $1);"
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
