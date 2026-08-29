package group

import (
	"database/sql"
	"expense-tracker/backend/types"
)

func (s *Store) GetGroupMemberByGroupID(groupID string) ([]*types.User, error) {
	query := "SELECT user_id FROM group_member WHERE group_id = $1 ORDER BY user_id ASC;"
	rowsGroup, err := s.db.Query(query, groupID)
	if err != nil {
		return nil, err
	}
	defer rowsGroup.Close()

	userIDs := make([]string, 0)
	for rowsGroup.Next() {
		var id string
		if err := rowsGroup.Scan(&id); err != nil {
			return nil, err
		}
		userIDs = append(userIDs, id)
	}
	if err := rowsGroup.Err(); err != nil {
		return nil, err
	}

	users := make([]*types.User, 0, len(userIDs))
	for _, id := range userIDs {
		user, err := s.getGroupMemberUser(id)
		if err != nil {
			return nil, err
		}
		if user != nil {
			users = append(users, user)
		}
	}

	return users, nil
}

func (s *Store) getGroupMemberUser(id string) (*types.User, error) {
	query := `
		SELECT id, username, firstname, lastname, email, password_hash,
			external_type, external_id, create_time_utc, is_active, nickname, role
		FROM users
		WHERE id = $1;
	`
	rows, err := s.db.Query(query, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var user *types.User
	for rows.Next() {
		user = new(types.User)
		var externalType sql.NullString
		var externalID sql.NullString
		if err := rows.Scan(
			&user.ID,
			&user.Username,
			&user.Firstname,
			&user.Lastname,
			&user.Email,
			&user.PasswordHashed,
			&externalType,
			&externalID,
			&user.CreateTime,
			&user.IsActive,
			&user.Nickname,
			&user.Role,
		); err != nil {
			return nil, err
		}
		user.ExternalType = nullStringToString(externalType)
		user.ExternalID = nullStringToString(externalID)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return user, nil
}

func nullStringToString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
