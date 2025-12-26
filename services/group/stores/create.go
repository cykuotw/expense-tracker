package group

import (
	"expense-tracker/types"

	"github.com/google/uuid"
)

func (s *Store) CreateGroup(group types.Group) error {
	// create group
	createTime := group.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	query := "INSERT INTO groups (" +
		"id, group_name, description, " +
		"create_time_utc, is_active, currency, create_by_user_id" +
		") VALUES (?, ?, ?, ?, ?, ?, ?);"

	_, err := s.db.Exec(query,
		group.ID, group.GroupName, group.Description,
		createTime, group.IsActive, group.Currency, group.CreateByUser)
	if err != nil {
		return err
	}

	// add user into group_member
	query = "INSERT INTO group_member (" +
		"id, group_id, user_id" +
		") VALUES (?, ?, ?);"
	_, err = s.db.Exec(query, uuid.NewString(), group.ID, group.CreateByUser)
	if err != nil {
		return err
	}

	return nil
}
