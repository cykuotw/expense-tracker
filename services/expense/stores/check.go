package store

import (
	"fmt"
)

func (s *Store) CheckExpenseExistByID(id string) (bool, error) {
	query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM expense WHERE id = '%s')", id)
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
