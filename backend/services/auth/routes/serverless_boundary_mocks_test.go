package route

import (
	"context"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"sync"
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
		ID:               uuid.New(),
		Username:         "test-user",
		Firstname:        "Test",
		Lastname:         "User",
		Email:            "user@example.test",
		Nickname:         "Test User",
		PasswordHashed:   string(passwordHash),
		HasLocalPassword: true,
		IsActive:         true,
		Role:             "user",
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
		s.usersByEmail[auth.NormalizeEmail(user.Email)] = user
	}
	if user.ExternalType != "" && user.ExternalID != "" {
		s.usersByExternal[user.ExternalType+":"+user.ExternalID] = user
	}
}

func (s *boundaryUserStore) GetUserByEmail(email string) (*types.User, error) {
	user, ok := s.usersByEmail[auth.NormalizeEmail(email)]
	if !ok {
		return nil, types.ErrUserNotExist
	}
	return &user, nil
}

type boundaryRegistrationStore struct {
	mu          sync.Mutex
	users       *boundaryUserStore
	invitations map[string]string
	used        map[string]bool
}

func newBoundaryRegistrationStore(users *boundaryUserStore) *boundaryRegistrationStore {
	return &boundaryRegistrationStore{
		users: users,
		invitations: map[string]string{
			"boundary-generic-invite": "",
			"boundary-email-invite":   "google-user@example.test",
		},
		used: make(map[string]bool),
	}
}

func (s *boundaryRegistrationStore) CreateInvitedUser(ctx context.Context, token string, user types.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	invitationEmail, exists := s.invitations[token]
	if !exists {
		return types.ErrInvitationInvalid
	}
	if s.used[token] {
		return types.ErrInvitationUsed
	}
	user.Email = auth.NormalizeEmail(user.Email)
	if invitationEmail != "" && auth.NormalizeEmail(invitationEmail) != user.Email {
		return types.ErrInvitationEmailMismatch
	}
	if _, exists := s.users.usersByEmail[user.Email]; exists {
		return types.ErrAccountConflict
	}
	if _, exists := s.users.usersByExternal[user.ExternalType+":"+user.ExternalID]; exists {
		return types.ErrAccountConflict
	}
	s.users.storeUser(user)
	s.used[token] = true
	return nil
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
