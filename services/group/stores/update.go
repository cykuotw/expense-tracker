package group

import (
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

	switch action {
	case "add":
		query := "INSERT INTO group_member (id, group_id, user_id) VALUES ($1, $2, $3)"
		_, err = s.db.Exec(query, uuid.NewString(), groupID, userID)
	case "delete":
		query := "DELETE FROM group_member WHERE group_id = $1 AND user_id = $2;"
		_, err = s.db.Exec(query, groupID, userID)
	}
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) UpdateGroupStatus(groupid string, isActive bool) error {
	query := "UPDATE groups SET is_active = $1 WHERE id = $2;"
	_, err := s.db.Exec(query, isActive, groupid)
	if err != nil {
		return err
	}
	return nil
}
