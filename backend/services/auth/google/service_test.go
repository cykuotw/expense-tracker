package google

import (
	"expense-tracker/backend/types"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestResolveUserFromClaims(t *testing.T) {
	t.Run("returns existing user for external identity match", func(t *testing.T) {
		userID := uuid.New()
		service := &Service{
			store: &mockUserStore{
				getUserByExternalIdentityFn: func(externalType string, externalID string) (*types.User, error) {
					assert.Equal(t, "google", externalType)
					assert.Equal(t, "google-sub-123", externalID)
					return &types.User{ID: userID, ExternalType: externalType, ExternalID: externalID}, nil
				},
				getUserByEmailFn: func(email string) (*types.User, error) {
					t.Fatalf("email lookup should not run")
					return nil, nil
				},
				createUserFn: func(user types.User) error {
					t.Fatalf("user should not be created")
					return nil
				},
			},
		}

		user, err := service.ResolveUserFromClaims(&types.VerifiedGoogleClaims{
			Subject: "google-sub-123",
		})

		assert.NoError(t, err)
		assert.Equal(t, userID, user.ID)
	})

	t.Run("requires an invitation for a verified unknown identity", func(t *testing.T) {
		service := &Service{
			store: &mockUserStore{
				getUserByExternalIdentityFn: func(externalType string, externalID string) (*types.User, error) {
					return nil, types.ErrUserNotExist
				},
				getUserByEmailFn: func(email string) (*types.User, error) {
					return nil, types.ErrUserNotExist
				},
				createUserFn: func(user types.User) error {
					t.Fatalf("login resolution must not create users")
					return nil
				},
			},
		}

		user, err := service.ResolveUserFromClaims(&types.VerifiedGoogleClaims{
			Subject:       "google-sub-123",
			Email:         "user@example.com",
			EmailVerified: boolPtr(true),
			GivenName:     "Taylor",
			FamilyName:    "Swift",
			Name:          "Taylor Swift",
		})

		assert.Nil(t, user)
		assert.ErrorIs(t, err, types.ErrInvitationRequired)
	})

	t.Run("returns conflict for existing email without external identity match", func(t *testing.T) {
		service := &Service{
			store: &mockUserStore{
				getUserByExternalIdentityFn: func(externalType string, externalID string) (*types.User, error) {
					return nil, types.ErrUserNotExist
				},
				getUserByEmailFn: func(email string) (*types.User, error) {
					return &types.User{ID: uuid.New(), Email: email}, nil
				},
				createUserFn: func(user types.User) error {
					t.Fatalf("user should not be created")
					return nil
				},
			},
		}

		user, err := service.ResolveUserFromClaims(&types.VerifiedGoogleClaims{
			Subject:       "google-sub-123",
			Email:         "user@example.com",
			EmailVerified: boolPtr(true),
		})

		assert.Nil(t, user)
		assert.ErrorIs(t, err, types.ErrAccountConflict)
	})

	t.Run("blocks unverified email", func(t *testing.T) {
		service := &Service{
			store: &mockUserStore{
				getUserByExternalIdentityFn: func(externalType string, externalID string) (*types.User, error) {
					return nil, types.ErrUserNotExist
				},
				getUserByEmailFn: func(email string) (*types.User, error) {
					return nil, types.ErrUserNotExist
				},
				createUserFn: func(user types.User) error {
					t.Fatalf("user should not be created")
					return nil
				},
			},
		}

		user, err := service.ResolveUserFromClaims(&types.VerifiedGoogleClaims{
			Subject:       "google-sub-123",
			Email:         "user@example.com",
			EmailVerified: boolPtr(false),
		})

		assert.Nil(t, user)
		assert.ErrorIs(t, err, types.ErrGoogleEmailNotVerified)
	})
}

func TestPrepareUserFromClaims(t *testing.T) {
	now := time.Unix(1700000000, 0).UTC()
	newID := uuid.New()
	service := &Service{
		store:   &mockUserStore{},
		now:     func() time.Time { return now },
		newUUID: func() uuid.UUID { return newID },
		hashSecret: func(value string) (string, error) {
			return "hashed-secret", nil
		},
	}

	user, err := service.PrepareUserFromClaims(&types.VerifiedGoogleClaims{
		Subject:       "google-sub-123",
		Email:         " User@Example.com ",
		EmailVerified: boolPtr(true),
		GivenName:     "Taylor",
		FamilyName:    "Swift",
	})

	assert.NoError(t, err)
	assert.Equal(t, newID, user.ID)
	assert.Equal(t, "user@example.com", user.Email)
	assert.Equal(t, "google", user.ExternalType)
	assert.Equal(t, "google-sub-123", user.ExternalID)
	assert.False(t, user.HasLocalPassword)
	assert.Equal(t, "hashed-secret", user.PasswordHashed)
	assert.Equal(t, now, user.CreateTime)
}

func TestNicknameFromClaimsFallbacks(t *testing.T) {
	assert.Equal(t, "Given", nicknameFromClaims(&types.VerifiedGoogleClaims{
		GivenName: "Given",
		Email:     "user@example.com",
	}))
	assert.Equal(t, "Display Name", nicknameFromClaims(&types.VerifiedGoogleClaims{
		Name:  "Display Name",
		Email: "user@example.com",
	}))
	assert.Equal(t, "local-part", nicknameFromClaims(&types.VerifiedGoogleClaims{
		Email: "local.part@example.com",
	}))
	assert.Equal(t, "google-user", nicknameFromClaims(nil))
}

func TestSanitizeUsername(t *testing.T) {
	assert.Equal(t, "hello-world", sanitizeUsername("Hello World"))
	assert.Equal(t, "abc_123", sanitizeUsername("abc_123"))
	assert.Equal(t, "google-user", sanitizeUsername("!!!"))
	assert.Equal(t, strings.Repeat("a", 32), sanitizeUsername(strings.Repeat("a", 40)))
}

func boolPtr(value bool) *bool {
	return &value
}
