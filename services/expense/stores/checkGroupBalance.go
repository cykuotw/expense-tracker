package store

import "fmt"

func (s *Store) CheckGroupBallanceAllSettled(groupId string) (bool, error) {
	query := fmt.Sprintf(
		"SELECT NOT EXISTS ("+
			"SELECT 1 FROM balance "+
			"WHERE group_id = '%s' AND is_settled = FALSE"+
			");",
		groupId,
	)
	rows, err := s.db.Query(query)
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

	return notExist, nil
}
