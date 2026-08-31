package group

import (
	"expense-tracker/backend/types"
	"github.com/google/uuid"
)

// ReplaceGroupMembers atomically replaces all non-creator group memberships.
func (s *Store) ReplaceGroupMembers(groupID, creatorID string, memberIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec("DELETE FROM group_member WHERE group_id = $1 AND user_id <> $2", groupID, creatorID); err != nil {
		return err
	}
	for _, memberID := range memberIDs {
		if _, err := tx.Exec(`INSERT INTO group_member (id, group_id, user_id)
			VALUES ($1, $2, $3) ON CONFLICT (group_id, user_id) DO NOTHING`, uuid.NewString(), groupID, memberID); err != nil {
			return err
		}
	}
	return tx.Commit()
}

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

func (s *Store) UpdateGroupStatus(groupID string, creatorID string, isActive bool) error {
	query := "UPDATE groups SET is_active = $1 WHERE id = $2 AND create_by_user_id = $3;"
	result, err := s.db.Exec(query, isActive, groupID, creatorID)
	if err != nil {
		return err
	}
	updated, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if updated == 0 {
		return types.ErrGroupNotExist
	}
	return nil
}
