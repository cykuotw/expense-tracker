package group

import (
	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

func (s *Store) CreateGroup(group types.Group, memberIDs []string) error {
	if group.GroupType == "" {
		group.GroupType = "home"
	}
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	createTime := group.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	query := "INSERT INTO groups (" +
		"id, group_name, description, " +
		"create_time_utc, is_active, currency, create_by_user_id, group_type" +
		") VALUES ($1, $2, $3, $4, $5, $6, $7, $8);"

	_, err = tx.Exec(query,
		group.ID, group.GroupName, group.Description,
		createTime, group.IsActive, group.Currency, group.CreateByUser, group.GroupType)
	if err != nil {
		return err
	}

	memberSet := make(map[string]struct{}, len(memberIDs)+1)
	memberSet[group.CreateByUser.String()] = struct{}{}
	for _, memberID := range memberIDs {
		memberSet[memberID] = struct{}{}
	}
	for memberID := range memberSet {
		if _, err := tx.Exec(
			"INSERT INTO group_member (id, group_id, user_id) VALUES ($1, $2, $3)",
			uuid.NewString(), group.ID, memberID,
		); err != nil {
			return err
		}
	}

	return tx.Commit()
}
