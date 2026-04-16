package route

import (
	"expense-tracker/backend/types"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

type boundaryUserStore struct {
	usersByID       map[string]types.User
	usersByEmail    map[string]types.User
	usersByExternal map[string]types.User
}

func newBoundaryUserStore(t *testing.T) *boundaryUserStore {
	t.Helper()

	passwordHash, err := bcrypt.GenerateFromPassword([]byte("testpassword"), bcrypt.DefaultCost)
	require.NoError(t, err)

	user := types.User{
		ID:             uuid.New(),
		Username:       "test-user",
		Firstname:      "Test",
		Lastname:       "User",
		Email:          "user@example.test",
		Nickname:       "Test User",
		PasswordHashed: string(passwordHash),
		IsActive:       true,
		Role:           "user",
	}

	store := &boundaryUserStore{
		usersByID:       make(map[string]types.User),
		usersByEmail:    make(map[string]types.User),
		usersByExternal: make(map[string]types.User),
	}
	store.storeUser(user)
	return store
}

func (s *boundaryUserStore) storeUser(user types.User) {
	s.usersByID[user.ID.String()] = user
	if user.Email != "" {
		s.usersByEmail[user.Email] = user
	}
	if user.ExternalType != "" && user.ExternalID != "" {
		s.usersByExternal[user.ExternalType+":"+user.ExternalID] = user
	}
}

func (s *boundaryUserStore) GetUserByEmail(email string) (*types.User, error) {
	user, ok := s.usersByEmail[email]
	if !ok {
		return nil, types.ErrUserNotExist
	}
	return &user, nil
}

func (s *boundaryUserStore) GetUserByExternalIdentity(externalType string, externalID string) (*types.User, error) {
	user, ok := s.usersByExternal[externalType+":"+externalID]
	if !ok {
		return nil, types.ErrUserNotExist
	}
	return &user, nil
}

func (s *boundaryUserStore) GetUserByID(id string) (*types.User, error) {
	user, ok := s.usersByID[id]
	if !ok {
		return nil, types.ErrUserNotExist
	}
	return &user, nil
}

func (s *boundaryUserStore) GetUsernameByID(userid string) (string, error) {
	user, ok := s.usersByID[userid]
	if !ok {
		return "", types.ErrUserNotExist
	}
	return user.Username, nil
}

func (s *boundaryUserStore) CreateUser(user types.User) error {
	s.storeUser(user)
	return nil
}

func (s *boundaryUserStore) CheckEmailExist(email string) (bool, error) {
	_, ok := s.usersByEmail[email]
	return ok, nil
}

func (s *boundaryUserStore) CheckUserExistByEmail(email string) (bool, error) {
	_, ok := s.usersByEmail[email]
	return ok, nil
}

func (s *boundaryUserStore) CheckUserExistByID(id string) (bool, error) {
	_, ok := s.usersByID[id]
	return ok, nil
}

func (s *boundaryUserStore) CheckUserExistByUsername(username string) (bool, error) {
	for _, user := range s.usersByID {
		if user.Username == username {
			return true, nil
		}
	}
	return false, nil
}
