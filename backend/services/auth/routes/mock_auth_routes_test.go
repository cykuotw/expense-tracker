package route

import (
	"expense-tracker/backend/types"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type baseAuthUserStore struct {
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

func (m *baseAuthUserStore) GetUserByEmail(email string) (*types.User, error) {
	if m.GetUserByEmailFn != nil {
		return m.GetUserByEmailFn(email)
	}
	return nil, types.ErrUserNotExist
}
func (m *baseAuthUserStore) GetUserByUsername(username string) (*types.User, error) {
	if m.GetUserByUsernameFn != nil {
		return m.GetUserByUsernameFn(username)
	}
	return nil, types.ErrUserNotExist
}
func (m *baseAuthUserStore) GetUserByID(id string) (*types.User, error) {
	if m.GetUserByIDFn != nil {
		return m.GetUserByIDFn(id)
	}
	return nil, nil
}
func (m *baseAuthUserStore) CreateUser(user types.User) error {
	if m.CreateUserFn != nil {
		return m.CreateUserFn(user)
	}
	return nil
}
func (m *baseAuthUserStore) GetUsernameByID(userid string) (string, error) {
	if m.GetUsernameByIDFn != nil {
		return m.GetUsernameByIDFn(userid)
	}
	return "", nil
}
func (m *baseAuthUserStore) CheckEmailExist(email string) (bool, error) {
	if m.CheckEmailExistFn != nil {
		return m.CheckEmailExistFn(email)
	}
	return false, nil
}
func (m *baseAuthUserStore) CheckUserExistByEmail(email string) (bool, error) {
	if m.CheckUserExistByEmailFn != nil {
		return m.CheckUserExistByEmailFn(email)
	}
	return false, nil
}
func (m *baseAuthUserStore) CheckUserExistByID(id string) (bool, error) {
	if m.CheckUserExistByIDFn != nil {
		return m.CheckUserExistByIDFn(id)
	}
	return false, nil
}
func (m *baseAuthUserStore) CheckUserExistByUsername(username string) (bool, error) {
	if m.CheckUserExistByUserFn != nil {
		return m.CheckUserExistByUserFn(username)
	}
	return false, nil
}

func loginUserStoreMock() *baseAuthUserStore {
	return &baseAuthUserStore{
		GetUserByEmailFn: func(email string) (*types.User, error) {
			hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
			return &types.User{PasswordHashed: string(hash)}, nil
		},
		GetUserByUsernameFn: func(username string) (*types.User, error) {
			hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
			return &types.User{PasswordHashed: string(hash)}, nil
		},
	}
}

func registerUserStoreMock() *baseAuthUserStore {
	return &baseAuthUserStore{
		GetUserByEmailFn:    func(email string) (*types.User, error) { return nil, types.ErrUserNotExist },
		GetUserByUsernameFn: func(username string) (*types.User, error) { return nil, types.ErrUserNotExist },
	}
}

type baseInvitationStore struct {
	CreateInvitationFn  func(invitation types.Invitation) error
	GetInvitationByTokenFn func(token string) (*types.Invitation, error)
	MarkInvitationUsedFn func(token string, email string) error
	ExpireInvitationFn   func(token string) error
	GetInvitationsFn     func() ([]types.Invitation, error)
}

func (m *baseInvitationStore) CreateInvitation(invitation types.Invitation) error {
	if m.CreateInvitationFn != nil {
		return m.CreateInvitationFn(invitation)
	}
	return nil
}
func (m *baseInvitationStore) GetInvitationByToken(token string) (*types.Invitation, error) {
	if m.GetInvitationByTokenFn != nil {
		return m.GetInvitationByTokenFn(token)
	}
	return &types.Invitation{Token: token, ExpiresAt: time.Now().Add(1 * time.Hour)}, nil
}
func (m *baseInvitationStore) MarkInvitationUsed(token string, email string) error {
	if m.MarkInvitationUsedFn != nil {
		return m.MarkInvitationUsedFn(token, email)
	}
	return nil
}
func (m *baseInvitationStore) ExpireInvitation(token string) error {
	if m.ExpireInvitationFn != nil {
		return m.ExpireInvitationFn(token)
	}
	return nil
}
func (m *baseInvitationStore) GetInvitations() ([]types.Invitation, error) {
	if m.GetInvitationsFn != nil {
		return m.GetInvitationsFn()
	}
	return []types.Invitation{}, nil
}

func invitationStoreMock() *baseInvitationStore {
	return &baseInvitationStore{}
}
