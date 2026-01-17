package group

import (
	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

type mockCreateGroupStore struct{}

func (m *mockCreateGroupStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockCreateGroupStore) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}
func (s *mockCreateGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	return nil, nil
}
func (m *mockCreateGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockCreateGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockCreateGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockCreateGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockCreateGroupStore) GetGroupCurrency(groupID string) (string, error) {
	return "", nil
}
func (m *mockCreateGroupStore) GetRelatedUser(currentUser string, groupId string) ([]*types.RelatedMember, error) {
	return nil, nil
}
func (m *mockCreateGroupStore) CheckGroupExistById(id string) (bool, error) {
	return false, nil
}
func (m *mockCreateGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	return false, nil
}

type mockCreateGroupUserStore struct{}

func (m *mockCreateGroupUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockCreateGroupUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockCreateGroupUserStore) GetUserByID(id string) (*types.User, error) {
	user := types.User{
		ID: mockUserId,
	}
	return &user, nil
}
func (m *mockCreateGroupUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockCreateGroupUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockCreateGroupUserStore) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
func (m *mockCreateGroupUserStore) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}
func (m *mockCreateGroupUserStore) CheckUserExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockCreateGroupUserStore) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}

type mockGetGroupStore struct{}

func (m *mockGetGroupStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockGetGroupStore) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}
func (s *mockGetGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	if userID != mockUserId.String() {
		return nil, types.ErrUserNotExist
	}
	if groupID != mockGroupId.String() {
		return nil, types.ErrGroupNotExist
	}

	return &types.Group{}, nil
}
func (m *mockGetGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockGetGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	users := []*types.User{
		{
			ID:       mockUserId,
			Username: uuid.New().String(),
		},
	}
	for i := 1; i < mockMemberNum; i++ {
		user := types.User{
			ID:       uuid.New(),
			Username: uuid.New().String(),
		}
		users = append(users, &user)
	}
	return users, nil
}
func (m *mockGetGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockGetGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockGetGroupStore) GetGroupCurrency(groupID string) (string, error) {
	return "", nil
}
func (m *mockGetGroupStore) GetRelatedUser(currentUser string, groupId string) ([]*types.RelatedMember, error) {
	return nil, nil
}
func (m *mockGetGroupStore) CheckGroupExistById(id string) (bool, error) {
	return false, nil
}
func (m *mockGetGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	return false, nil
}

type mockGetGroupUserStore struct{}

func (m *mockGetGroupUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetGroupUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetGroupUserStore) GetUserByID(id string) (*types.User, error) {
	user := types.User{
		ID: mockUserId,
	}
	return &user, nil
}
func (m *mockGetGroupUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockGetGroupUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockGetGroupUserStore) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
func (m *mockGetGroupUserStore) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}
func (m *mockGetGroupUserStore) CheckUserExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockGetGroupUserStore) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}

type mockGetGroupListStore struct{}

func (m *mockGetGroupListStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockGetGroupListStore) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}
func (s *mockGetGroupListStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	return nil, nil
}
func (m *mockGetGroupListStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	if userid != mockUserId.String() {
		return nil, nil
	}
	var groups []*types.Group

	for i := 0; i < mockGroupListLen; i++ {
		group := types.Group{
			ID:        uuid.New(),
			GroupName: uuid.New().String(),
		}
		groups = append(groups, &group)
	}
	return groups, nil
}
func (m *mockGetGroupListStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockGetGroupListStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockGetGroupListStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockGetGroupListStore) GetGroupCurrency(groupID string) (string, error) {
	return "", nil
}
func (m *mockGetGroupListStore) GetRelatedUser(currentUser string, groupId string) ([]*types.RelatedMember, error) {
	return nil, nil
}
func (m *mockGetGroupListStore) CheckGroupExistById(id string) (bool, error) {
	return false, nil
}
func (m *mockGetGroupListStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	return false, nil
}

type mockGetGroupListUserStore struct{}

func (m *mockGetGroupListUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetGroupListUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetGroupListUserStore) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetGroupListUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockGetGroupListUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockGetGroupListUserStore) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
func (m *mockGetGroupListUserStore) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}
func (m *mockGetGroupListUserStore) CheckUserExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockGetGroupListUserStore) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}

type mockUpdateGroupMemberStore struct{}

func (m *mockUpdateGroupMemberStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockUpdateGroupMemberStore) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}
func (s *mockUpdateGroupMemberStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	return nil, nil
}
func (m *mockUpdateGroupMemberStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockUpdateGroupMemberStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockUpdateGroupMemberStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockUpdateGroupMemberStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockUpdateGroupMemberStore) GetGroupCurrency(groupID string) (string, error) {
	return "", nil
}
func (m *mockUpdateGroupMemberStore) GetRelatedUser(currentUser string, groupId string) ([]*types.RelatedMember, error) {
	return nil, nil
}
func (m *mockUpdateGroupMemberStore) CheckGroupExistById(id string) (bool, error) {
	if id != mockGroupId.String() {
		return false, nil
	}
	return true, nil
}
func (m *mockUpdateGroupMemberStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	if groupId != mockGroupId.String() {
		return false, nil
	}
	if userId != mockRequesterId.String() {
		return false, nil
	}
	return true, nil
}

type mockUpdateGroupMemberUserStore struct{}

func (m *mockUpdateGroupMemberUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockUpdateGroupMemberUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockUpdateGroupMemberUserStore) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}
func (m *mockUpdateGroupMemberUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockUpdateGroupMemberUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockUpdateGroupMemberUserStore) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
func (m *mockUpdateGroupMemberUserStore) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}
func (m *mockUpdateGroupMemberUserStore) CheckUserExistByID(id string) (bool, error) {
	if id != mockUserId.String() {
		return false, nil
	}
	return true, nil
}
func (m *mockUpdateGroupMemberUserStore) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}

type mockArchiveGroupStore struct{}

func (m *mockArchiveGroupStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockArchiveGroupStore) GetGroupByID(id string) (*types.Group, error) {
	if id != mockGroupId.String() {
		return nil, types.ErrGroupNotExist
	}
	return nil, nil
}
func (s *mockArchiveGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	return nil, nil
}
func (m *mockArchiveGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockArchiveGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockArchiveGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockArchiveGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockArchiveGroupStore) GetGroupCurrency(groupID string) (string, error) {
	return "", nil
}
func (m *mockArchiveGroupStore) GetRelatedUser(currentUser string, groupId string) ([]*types.RelatedMember, error) {
	return nil, nil
}
func (m *mockArchiveGroupStore) CheckGroupExistById(id string) (bool, error) {
	return false, nil
}
func (m *mockArchiveGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	return false, nil
}

type mockArchiveGroupUserStore struct{}

func (m *mockArchiveGroupUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockArchiveGroupUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockArchiveGroupUserStore) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}
func (m *mockArchiveGroupUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockArchiveGroupUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockArchiveGroupUserStore) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
func (m *mockArchiveGroupUserStore) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}
func (m *mockArchiveGroupUserStore) CheckUserExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockArchiveGroupUserStore) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}
