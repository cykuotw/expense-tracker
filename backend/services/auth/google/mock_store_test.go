package google

import "expense-tracker/backend/types"

type mockUserStore struct {
	getUserByExternalIdentityFn func(externalType string, externalID string) (*types.User, error)
	getUserByEmailFn            func(email string) (*types.User, error)
	createUserFn                func(user types.User) error
}

func (f *mockUserStore) GetUserByExternalIdentity(externalType string, externalID string) (*types.User, error) {
	return f.getUserByExternalIdentityFn(externalType, externalID)
}

func (f *mockUserStore) GetUserByEmail(email string) (*types.User, error) {
	return f.getUserByEmailFn(email)
}

func (f *mockUserStore) CreateUser(user types.User) error {
	return f.createUserFn(user)
}

func (f *mockUserStore) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}

func (f *mockUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}

func (f *mockUserStore) CheckEmailExist(email string) (bool, error) {
	return false, nil
}

func (f *mockUserStore) CheckUserExistByEmail(email string) (bool, error) {
	return false, nil
}

func (f *mockUserStore) CheckUserExistByID(id string) (bool, error) {
	return false, nil
}

func (f *mockUserStore) CheckUserExistByUsername(username string) (bool, error) {
	return false, nil
}
