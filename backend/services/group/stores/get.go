package group

import (
	"database/sql"
	"errors"

	"expense-tracker/backend/services/user"
	"expense-tracker/backend/types"
)

func (s *Store) GetGroupByID(id string) (*types.Group, error) {
	query := "SELECT id, group_name, description, create_time_utc, is_active, create_by_user_id, currency, group_type FROM groups WHERE id = $1;"
	group, err := scanRowIntoGroup(s.db.QueryRow(query, id))
	if errors.Is(err, sql.ErrNoRows) {
		return nil, types.ErrGroupNotExist
	}
	if err != nil {
		return nil, err
	}

	return group, nil
}

func (s *Store) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	// check group id exist
	exist, err := s.CheckGroupExistById(groupID)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, types.ErrGroupNotExist
	}

	// check user id exist
	userStore := user.NewStore(s.db)
	exist, err = userStore.CheckUserExistByID(userID)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, types.ErrUserNotExist
	}

	// check user is group member
	exist, err = s.CheckGroupUserPairExist(groupID, userID)
	if err != nil {
		return nil, err
	}
	if !exist {
		return nil, types.ErrUserNotPermitted
	}

	// get group
	group, err := s.GetGroupByID(groupID)
	if err != nil {
		return nil, err
	}

	return group, nil
}
