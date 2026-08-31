package group

import "expense-tracker/backend/types"

func (s *Store) GetGroupMemberByGroupID(groupID string) ([]*types.User, error) {
	query := `
		SELECT
			u.id, u.username, u.firstname, u.lastname, u.email, u.password_hash,
			COALESCE(u.external_type, ''), COALESCE(u.external_id, ''),
			u.create_time_utc, u.is_active, u.nickname, u.role
		FROM group_member gm
		INNER JOIN users u ON u.id = gm.user_id
		WHERE gm.group_id = $1
		ORDER BY u.id ASC;
	`
	rows, err := s.db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	users := make([]*types.User, 0)
	for rows.Next() {
		user := new(types.User)
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Firstname,
			&user.Lastname,
			&user.Email,
			&user.PasswordHashed,
			&user.ExternalType,
			&user.ExternalID,
			&user.CreateTime,
			&user.IsActive,
			&user.Nickname,
			&user.Role,
		); err != nil {
			return nil, err
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return users, nil
}
