package group

import (
	"expense-tracker/services/user"
	"expense-tracker/types"
	"fmt"

	"github.com/google/uuid"
)

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
		return nil, types.ErrGroupNotExist
	}

	return group, nil
}
