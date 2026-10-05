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
	if err := tx.QueryRow(`SELECT id FROM groups WHERE id = $1 AND create_by_user_id = $2 AND is_active = TRUE FOR UPDATE`, groupID, creatorID).Scan(&lockedGroupID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return types.ErrGroupNotExist
		}
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

	var active bool
	if err := tx.QueryRow(`SELECT is_active FROM groups WHERE id = $1 FOR UPDATE`, groupID).Scan(&active); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if action == "delete" {
				return tx.Commit()
			}
			return types.ErrGroupNotExist
		}
		return err
	}
	if !active {
		return types.ErrGroupNotExist
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
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var active bool
	err = tx.QueryRow(`SELECT is_active FROM groups WHERE id = $1 AND create_by_user_id = $2 FOR UPDATE`, groupID, creatorID).Scan(&active)
	if errors.Is(err, sql.ErrNoRows) {
		return types.ErrGroupNotExist
	}
	if err != nil {
		return err
	}
	if active == isActive {
		return tx.Commit()
	}
	if !isActive {
		var settled bool
		if err := tx.QueryRow(groupSettledSQL, groupID).Scan(&settled); err != nil {
			return err
		}
		if !settled {
			return types.ErrGroupUnsettled
		}
	}
	if _, err := tx.Exec(`UPDATE groups SET is_active = $1 WHERE id = $2`, isActive, groupID); err != nil {
		return err
	}
	return tx.Commit()
}

func (s *Store) UpdateGroup(group types.Group) error {
	result, err := s.db.Exec(`UPDATE groups
		SET group_name = $1, description = $2, group_type = $3
		WHERE id = $4 AND create_by_user_id = $5 AND is_active = TRUE`,
		group.GroupName, group.Description, group.GroupType, group.ID, group.CreateByUser)
	if err != nil {
		return err
	}
	count, err := result.RowsAffected()
	if err == nil && count == 0 {
		return types.ErrGroupNotExist
	}
	return err
}
