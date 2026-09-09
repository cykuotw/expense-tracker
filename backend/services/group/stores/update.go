package group

import (
	"database/sql"
	"errors"
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

	var lockedGroupID string
	if err := tx.QueryRow(`SELECT id FROM groups WHERE id = $1 FOR UPDATE`, groupID).Scan(&lockedGroupID); err != nil {
		return err
	}
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
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var lockedGroupID string
	if err := tx.QueryRow(`SELECT id FROM groups WHERE id = $1 FOR UPDATE`, groupID).Scan(&lockedGroupID); err != nil {
		if action == "delete" && errors.Is(err, sql.ErrNoRows) {
			return tx.Commit()
		}
		return err
	}

	var exist bool
	if err := tx.QueryRow(`SELECT EXISTS (
		SELECT 1 FROM group_member WHERE group_id = $1 AND user_id = $2
	)`, groupID, userID).Scan(&exist); err != nil {
		return err
	}

	// if exist in add mode
	// 	  OR
	// 	  not exist in delete mode
	// -> just return
	if (action == "add" && exist) || (action == "delete" && !exist) {
		return tx.Commit()
	}

	switch action {
	case "add":
		query := "INSERT INTO group_member (id, group_id, user_id) VALUES ($1, $2, $3)"
		_, err = tx.Exec(query, uuid.NewString(), groupID, userID)
	case "delete":
		query := "DELETE FROM group_member WHERE group_id = $1 AND user_id = $2;"
		_, err = tx.Exec(query, groupID, userID)
	}
	if err != nil {
		return err
	}

	return tx.Commit()
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

func (s *Store) UpdateGroup(group types.Group) error {
	_, err := s.db.Exec(`UPDATE groups
		SET group_name = $1, description = $2, group_type = $3
		WHERE id = $4 AND create_by_user_id = $5`,
		group.GroupName, group.Description, group.GroupType, group.ID, group.CreateByUser)
	return err
}
