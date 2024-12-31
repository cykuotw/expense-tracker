package group

import (
	"expense-tracker/types"
	"fmt"
)

func (s *Store) GetRelatedUser(currentUser string, groupId string) ([]*types.RelatedMember, error) {
	query := fmt.Sprintf(
		`WITH former_member AS (
			SELECT user_id
			FROM group_member
			WHERE group_id IN (
				SELECT group_id
				FROM group_member 
				WHERE user_id = '%s'))

		SELECT DISTINCT
			u.id, 
			u.username,
			CASE
				WHEN EXISTS (
					SELECT 1 
					FROM group_member as gm
					WHERE gm.user_id = u.id
						AND gm.group_id = '%s'
				) THEN TRUE
				ELSE FALSE
			END AS exist_in_group
		FROM users AS u
		JOIN former_member AS fm
		ON u.id = fm.user_id
		WHERE u.id <> '%s'
		ORDER BY u.username;`,
		currentUser, groupId, currentUser,
	)
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var members []*types.RelatedMember
	for rows.Next() {
		member := new(types.RelatedMember)
		err := rows.Scan(&member.UserID, &member.Username, &member.ExistInGroup)
		if err != nil {
			return nil, err
		}
		members = append(members, member)
	}

	return members, nil
}
