package route

import (
	"context"
	"expense-tracker/backend/types"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type baseAuthUserStore struct {
	GetUserByEmailFn        func(email string) (*types.User, error)
	GetUserByExternalIDFn   func(externalType string, externalID string) (*types.User, error)
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
func (m *baseAuthUserStore) GetUserByExternalIdentity(externalType string, externalID string) (*types.User, error) {
	if m.GetUserByExternalIDFn != nil {
		return m.GetUserByExternalIDFn(externalType, externalID)
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
	return &types.User{IsActive: true}, nil
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
			return &types.User{PasswordHashed: string(hash), HasLocalPassword: true, IsActive: true}, nil
		},
		GetUserByUsernameFn: func(username string) (*types.User, error) {
			hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
			return &types.User{PasswordHashed: string(hash), HasLocalPassword: true, IsActive: true}, nil
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
	CreateInvitationFn   func(invitation types.Invitation) error
	ExchangeInvitationFn func(token string, registrationSession string) (*types.Invitation, error)
}

func (m *baseInvitationStore) CreateInvitation(invitation types.Invitation) error {
	if m.CreateInvitationFn != nil {
		return m.CreateInvitationFn(invitation)
	}
	return nil
}
func (m *baseInvitationStore) ExchangeInvitation(token string, registrationSession string) (*types.Invitation, error) {
	if m.ExchangeInvitationFn != nil {
		return m.ExchangeInvitationFn(token, registrationSession)
	}
	return &types.Invitation{Email: "", ExpiresAt: time.Now().Add(time.Hour)}, nil
}

func invitationStoreMock() *baseInvitationStore {
	return &baseInvitationStore{}
}

type baseRegistrationStore struct {
	CreateInvitedUserFn func(ctx context.Context, token string, user types.User) error
}

func (m *baseRegistrationStore) CreateInvitedUser(ctx context.Context, token string, user types.User) error {
	if m.CreateInvitedUserFn != nil {
		return m.CreateInvitedUserFn(ctx, token, user)
	}
	return nil
}

func registrationStoreMock() *baseRegistrationStore {
	return &baseRegistrationStore{}
}
