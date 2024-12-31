package group

import (
	"expense-tracker/types"
	"fmt"

	"github.com/google/uuid"
)

func (s *Store) GetGroupListByUser(userID string) ([]*types.Group, error) {
	// get group id where user is member
	query := fmt.Sprintf("SELECT group_id FROM group_member WHERE user_id='%s';", userID)
	rowsMember, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rowsMember.Close()

	var groupIds []string
	for rowsMember.Next() {
		var id string
		err := rowsMember.Scan(&id)
		if err != nil {
			return nil, err
		}
		groupIds = append(groupIds, id)
	}

	// get group details
	var groups []*types.Group

	for _, id := range groupIds {
		query = fmt.Sprintf("SELECT * FROM groups WHERE id='%s';", id)
		rows, err := s.db.Query(query)
		if err != nil {
			return nil, err
		}
		defer rows.Close()

		for rows.Next() {
			group, err := scanRowIntoGroup(rows)
			if err != nil {
				return nil, err
			}
			if group.ID == uuid.Nil {
				continue
			}
			groups = append(groups, group)
		}
	}

	return groups, nil
}
