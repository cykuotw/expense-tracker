package group

import (
	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

// base group store

type mockGroupStore struct {
	CreateGroupFn          func(group types.Group) error
	GetGroupByIDFn         func(id string) (*types.Group, error)
	GetGroupByIDAndUserFn  func(groupID string, userID string) (*types.Group, error)
	GetGroupListByUserFn   func(userid string) ([]*types.Group, error)
	GetGroupMemberByGroupIDFn func(groupId string) ([]*types.User, error)
	UpdateGroupMemberFn    func(action string, userid string, groupid string) error
	UpdateGroupStatusFn    func(groupid string, isActive bool) error
	GetGroupCurrencyFn     func(groupID string) (string, error)
	GetRelatedUserFn       func(currentUser string, groupId string) ([]*types.RelatedMember, error)
	CheckGroupExistByIdFn  func(id string) (bool, error)
	CheckGroupUserPairExistFn func(groupId string, userId string) (bool, error)
}

func (m *mockGroupStore) CreateGroup(group types.Group) error {
	if m.CreateGroupFn != nil {
		return m.CreateGroupFn(group)
	}
	return nil
}
func (m *mockGroupStore) GetGroupByID(id string) (*types.Group, error) {
	if m.GetGroupByIDFn != nil {
		return m.GetGroupByIDFn(id)
	}
	return nil, nil
}
func (m *mockGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	if m.GetGroupByIDAndUserFn != nil {
		return m.GetGroupByIDAndUserFn(groupID, userID)
	}
	return nil, nil
}
func (m *mockGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	if m.GetGroupListByUserFn != nil {
		return m.GetGroupListByUserFn(userid)
	}
	return nil, nil
}
func (m *mockGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	if m.GetGroupMemberByGroupIDFn != nil {
		return m.GetGroupMemberByGroupIDFn(groupId)
	}
	return nil, nil
}
func (m *mockGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	if m.UpdateGroupMemberFn != nil {
		return m.UpdateGroupMemberFn(action, userid, groupid)
	}
	return nil
}
func (m *mockGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	if m.UpdateGroupStatusFn != nil {
		return m.UpdateGroupStatusFn(groupid, isActive)
	}
	return nil
}
func (m *mockGroupStore) GetGroupCurrency(groupID string) (string, error) {
	if m.GetGroupCurrencyFn != nil {
		return m.GetGroupCurrencyFn(groupID)
	}
	return "", nil
}
func (m *mockGroupStore) GetRelatedUser(currentUser string, groupId string) ([]*types.RelatedMember, error) {
	if m.GetRelatedUserFn != nil {
		return m.GetRelatedUserFn(currentUser, groupId)
	}
	return nil, nil
}
func (m *mockGroupStore) CheckGroupExistById(id string) (bool, error) {
	if m.CheckGroupExistByIdFn != nil {
		return m.CheckGroupExistByIdFn(id)
	}
	return false, nil
}
func (m *mockGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	if m.CheckGroupUserPairExistFn != nil {
		return m.CheckGroupUserPairExistFn(groupId, userId)
	}
	return false, nil
}

// base user store

type mockUserStore struct {
	GetUserByEmailFn        func(email string) (*types.User, error)
	GetUserByUsernameFn     func(username string) (*types.User, error)
	GetUserByIDFn           func(id string) (*types.User, error)
	CreateUserFn            func(user types.User) error
	GetUsernameByIDFn       func(userid string) (string, error)
	CheckEmailExistFn       func(email string) (bool, error)
	CheckUserExistByEmailFn func(email string) (bool, error)
	CheckUserExistByIDFn    func(id string) (bool, error)
	CheckUserExistByUserFn  func(username string) (bool, error)
}

func (m *mockUserStore) GetUserByEmail(email string) (*types.User, error) {
	if m.GetUserByEmailFn != nil {
		return m.GetUserByEmailFn(email)
	}
	return nil, nil
}
func (m *mockUserStore) GetUserByUsername(username string) (*types.User, error) {
	if m.GetUserByUsernameFn != nil {
		return m.GetUserByUsernameFn(username)
	}
	return nil, nil
}
func (m *mockUserStore) GetUserByID(id string) (*types.User, error) {
	if m.GetUserByIDFn != nil {
		return m.GetUserByIDFn(id)
	}
	return nil, nil
}
func (m *mockUserStore) CreateUser(user types.User) error {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(user)
	}
	return nil
}
func (m *mockUserStore) GetUsernameByID(userid string) (string, error) {
	if m.GetUsernameByIDFn != nil {
		return m.GetUsernameByIDFn(userid)
	}
	return "", nil
}
func (m *mockUserStore) CheckEmailExist(email string) (bool, error) {
	if m.CheckEmailExistFn != nil {
		return m.CheckEmailExistFn(email)
	}
	return false, nil
}
func (m *mockUserStore) CheckUserExistByEmail(email string) (bool, error) {
	if m.CheckUserExistByEmailFn != nil {
		return m.CheckUserExistByEmailFn(email)
	}
	return false, nil
}
func (m *mockUserStore) CheckUserExistByID(id string) (bool, error) {
	if m.CheckUserExistByIDFn != nil {
		return m.CheckUserExistByIDFn(id)
	}
	return false, nil
}
func (m *mockUserStore) CheckUserExistByUsername(username string) (bool, error) {
	if m.CheckUserExistByUserFn != nil {
		return m.CheckUserExistByUserFn(username)
	}
	return false, nil
}

func groupStoreMock() *mockGroupStore { return &mockGroupStore{} }
func userStoreMock() *mockUserStore   { return &mockUserStore{} }

func createGroupStoreMock() *mockGroupStore { return groupStoreMock() }

func createGroupUserStoreMock() *mockUserStore {
	store := userStoreMock()
	store.GetUserByIDFn = func(id string) (*types.User, error) {
		return &types.User{ID: mockUserId}, nil
	}
	return store
}

func getGroupStoreMock() *mockGroupStore {
	store := groupStoreMock()
	store.GetGroupByIDAndUserFn = func(groupID string, userID string) (*types.Group, error) {
		if userID != mockUserId.String() {
			return nil, types.ErrUserNotExist
		}
		if groupID != mockGroupId.String() {
			return nil, types.ErrGroupNotExist
		}
		return &types.Group{}, nil
	}
	store.GetGroupMemberByGroupIDFn = func(groupId string) ([]*types.User, error) {
		users := []*types.User{{ID: mockUserId, Username: uuid.New().String()}}
		for i := 1; i < mockMemberNum; i++ {
			user := types.User{ID: uuid.New(), Username: uuid.New().String()}
			users = append(users, &user)
		}
		return users, nil
	}
	return store
}

func getGroupUserStoreMock() *mockUserStore {
	store := userStoreMock()
	store.GetUserByIDFn = func(id string) (*types.User, error) {
		return &types.User{ID: mockUserId}, nil
	}
	return store
}

func getGroupListStoreMock() *mockGroupStore {
	store := groupStoreMock()
	store.GetGroupListByUserFn = func(userid string) ([]*types.Group, error) {
		if userid != mockUserId.String() {
			return nil, nil
		}
		groups := make([]*types.Group, 0, mockGroupListLen)
		for i := 0; i < mockGroupListLen; i++ {
			group := types.Group{ID: uuid.New(), GroupName: uuid.New().String()}
			groups = append(groups, &group)
		}
		return groups, nil
	}
	return store
}

func getGroupListUserStoreMock() *mockUserStore {
	store := userStoreMock()
	store.GetUserByIDFn = func(id string) (*types.User, error) {
		return &types.User{ID: mockUserId}, nil
	}
	return store
}

func updateGroupMemberStoreMock() *mockGroupStore {
	store := groupStoreMock()
	store.CheckGroupExistByIdFn = func(id string) (bool, error) {
		return id == mockGroupId.String(), nil
	}
	store.CheckGroupUserPairExistFn = func(groupId string, userId string) (bool, error) {
		return groupId == mockGroupId.String() && userId == mockRequesterId.String(), nil
	}
	return store
}

func updateGroupMemberUserStoreMock() *mockUserStore {
	store := userStoreMock()
	store.CheckUserExistByIDFn = func(id string) (bool, error) {
		return id == mockUserId.String(), nil
	}
	return store
}

func archiveGroupStoreMock() *mockGroupStore {
	store := groupStoreMock()
	store.GetGroupByIDFn = func(id string) (*types.Group, error) {
		if id != mockGroupId.String() {
			return nil, types.ErrGroupNotExist
		}
		return &types.Group{}, nil
	}
	store.UpdateGroupStatusFn = func(groupid string, isActive bool) error { return nil }
	return store
}

func archiveGroupUserStoreMock() *mockUserStore { return userStoreMock() }
