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

	var userIDs []string
	for rowsGroup.Next() {
		var id string
		rowsGroup.Scan(&id)
		userIDs = append(userIDs, id)
	}

	var users []*types.User
	for _, id := range userIDs {
		query := "SELECT * FROM users WHERE id = $1;"
		rows, err := s.db.Query(query, id)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			user := new(types.User)
			var externalType sql.NullString
			var externalID sql.NullString
			err := rows.Scan(
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
			)
			if err != nil {
				return nil, err
			}
			user.ExternalType = nullStringToString(externalType)
			user.ExternalID = nullStringToString(externalID)
			users = append(users, user)
		}
	}

	return users, nil
}

func nullStringToString(value sql.NullString) string {
	if !value.Valid {
		return ""
	}
	return value.String
}
