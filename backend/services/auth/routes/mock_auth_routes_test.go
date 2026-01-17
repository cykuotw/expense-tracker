package route

import (
	"expense-tracker/backend/types"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type mockStoreLogin struct{}

func (m *mockStoreLogin) GetUserByEmail(email string) (*types.User, error) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
	user := types.User{
		PasswordHashed: string(hash),
	}
	return &user, nil
}
func (m *mockStoreLogin) GetUserByUsername(username string) (*types.User, error) {
	hash, _ := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
	user := types.User{
		PasswordHashed: string(hash),
	}
	return &user, nil
}
func (m *mockStoreLogin) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}
func (m *mockStoreLogin) CreateUser(user types.User) error {
	return nil
}
func (m *mockStoreLogin) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockStoreLogin) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
func (m *mockStoreLogin) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}
func (m *mockStoreLogin) CheckUserExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockStoreLogin) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}

type mockStoreRegister struct{}

func (m *mockStoreRegister) GetUserByEmail(email string) (*types.User, error) {
	return nil, types.ErrUserNotExist
}
func (m *mockStoreRegister) GetUserByUsername(username string) (*types.User, error) {
	return nil, types.ErrUserNotExist
}
func (m *mockStoreRegister) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}
func (m *mockStoreRegister) CreateUser(user types.User) error {
	return nil
}
func (m *mockStoreRegister) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockStoreRegister) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
func (m *mockStoreRegister) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}
func (m *mockStoreRegister) CheckUserExistByID(id string) (bool, error) {
	return false, nil
}
func (m *mockStoreRegister) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}

type mockInvitationStore struct{}

func (m *mockInvitationStore) CreateInvitation(invitation types.Invitation) error {
	return nil
}
func (m *mockInvitationStore) GetInvitationByToken(token string) (*types.Invitation, error) {
	return &types.Invitation{
		Token:     token,
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}, nil
}
func (m *mockInvitationStore) MarkInvitationUsed(token string, email string) error {
	return nil
}
func (m *mockInvitationStore) ExpireInvitation(token string) error {
	return nil
}
func (m *mockInvitationStore) GetInvitations() ([]types.Invitation, error) {
	return []types.Invitation{}, nil
}
