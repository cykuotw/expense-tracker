package group

import (
	"database/sql"
	"expense-tracker/types"
	"fmt"

	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateGroup(group types.Group) error {
	// create group
	createTime := group.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	query := fmt.Sprintf(
		"INSERT INTO groups ("+
			"id, group_name, description, "+
			"create_time_utc, is_active, currency, create_by_user_id"+
			") VALUES ('%s', '%s', '%s', '%s', '%t', '%s', '%s');",
		group.ID, group.GroupName, group.Description,
		createTime, group.IsActive, group.Currency, group.CreateByUser,
	)

	_, err := s.db.Exec(query)
	if err != nil {
		return err
	}

	// add user into group_member
	query = fmt.Sprintf(
		"INSERT INTO group_member ("+
			"id, group_id, user_id"+
			") VALUES ('%s', '%s', '%s');",
		uuid.NewString(), group.ID, group.CreateByUser,
	)
	_, err = s.db.Exec(query)
	if err != nil {
		return err
	}

	return nil
}

func (s *Store) GetGroupByID(id string) (*types.Group, error) {
	query := fmt.Sprintf("SELECT * FROM groups WHERE id='%s';", id)
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	group := new(types.Group)
	for rows.Next() {
		group, err = scanRowIntoGroup(rows)
		if err != nil {
			return nil, err
		}
	}

	if group.ID == uuid.Nil {
		return nil, types.ErrGroupNotExist
	}

	return group, nil
}

func (s *Store) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	// check group id exist
	group, err := s.GetGroupByID(groupID)
	if err != nil {
		return nil, types.ErrGroupNotExist
	}
	// check user id exist
	query := fmt.Sprintf("SELECT COUNT(*) FROM users WHERE id = '%s';", userID)
	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}
	var count int
	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return nil, err
		}
	}
	rows.Close()
	if count != 1 {
		return nil, types.ErrUserNotExist
	}

	// check user is group member
	query = fmt.Sprintf(
		"SELECT COUNT(*) FROM group_member WHERE group_id='%s' "+
			"AND user_id='%s';",
		groupID, userID,
	)
	rows, err = s.db.Query(query)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		if err := rows.Scan(&count); err != nil {
			return nil, err
		}
	}
	rows.Close()
	if count != 1 {
		return nil, types.ErrUserNotPermitted
	}

	return group, nil
}

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

func (s *Store) GetGroupMemberByGroupID(groupID string) ([]*types.User, error) {
	query := fmt.Sprintf(
		"SELECT user_id FROM group_member WHERE group_id='%s';",
		groupID)
	rowsGroup, err := s.db.Query(query)
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
		query := fmt.Sprintf("SELECT * FROM users WHERE id='%s';", id)
		rows, err := s.db.Query(query)
		if err != nil {
			return nil, err
		}
		for rows.Next() {
			user := new(types.User)
			err := rows.Scan(
				&user.ID,
				&user.Username,
				&user.Firstname,
				&user.Lastname,
				&user.Email,
				&user.PasswordHashed,
				&user.ExternalType,
				&user.ExternalID,
				&user.CreateTime,
				&user.IsActive,
			)
			if err != nil {
				return nil, err
			}
			users = append(users, user)
		}
	}

	return users, nil
}

func (s *Store) UpdateGroupMember(action string, userID string, groupID string) error {
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
	_, err := s.db.Exec(query)
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

func scanRowIntoGroup(rows *sql.Rows) (*types.Group, error) {
	group := new(types.Group)

	err := rows.Scan(
		&group.ID,
		&group.GroupName,
		&group.Description,
		&group.CreateTime,
		&group.IsActive,
		&group.CreateByUser,
	)
	if err != nil {
		return nil, err
	}
	return group, nil
}
