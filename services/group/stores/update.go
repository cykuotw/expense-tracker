package group

import (
	"fmt"

	"github.com/google/uuid"
)

func (s *Store) UpdateGroupMember(action string, userID string, groupID string) error {
	// check userID and groupID pair exist,
	exist, err := s.CheckGroupUserPairExist(groupID, userID)
	if err != nil {
		return err
	}

	// if exist in add mode
	// 	  OR
	// 	  not exist in delete mode
	// -> just return
	if (action == "add" && exist) || (action == "delete" && !exist) {
		return nil
	}

	query := ""
	if action == "add" {
		query = fmt.Sprintf(
			"INSERT INTO group_member ("+
				"id, group_id, user_id"+
				") VALUES ('%s', '%s', '%s')",
			uuid.NewString(), groupID, userID,
		)
	} else if action == "delete" {
		query = fmt.Sprintf(
			"DELETE FROM group_member WHERE group_id='%s' AND user_id='%s';",
			groupID, userID,
		)
	}
	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) UpdateGroupStatus(groupid string, isActive bool) error {
	query := fmt.Sprintf("UPDATE groups SET is_active='%t' WHERE id='%s';",
		isActive, groupid)
	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}
	return nil
}
