package user_test

import (
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/services/user"
	"expense-tracker/backend/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestChangeOwnPasswordPreservesCurrentRefreshSession(t *testing.T) {
	db := openTestDB(t)
	userID := uuid.New()
	oldHash, err := auth.HashPassword("old-password")
	assert.NoError(t, err)
	insertUser(db, types.User{
		ID: userID, Username: "account-test", Firstname: "Account", Lastname: "Test",
		Email: "account-" + userID.String()[:8] + "@example.com", PasswordHashed: oldHash,
		CreateTime: time.Now(), IsActive: true,
	})
	defer cleanUser(db, userID)

	refreshStore := auth.NewRefreshStore(db)
	currentID := uuid.New()
	otherID := uuid.New()
	for _, id := range []uuid.UUID{currentID, otherID} {
		assert.NoError(t, refreshStore.CreateRefreshToken(types.RefreshToken{
			ID: id, UserID: userID, TokenHash: "hash-" + id.String(),
			ExpiresAt: time.Now().Add(time.Hour), CreatedAt: time.Now(),
		}))
		defer func(tokenID uuid.UUID) {
			_, _ = db.Exec("DELETE FROM refresh_tokens WHERE id = $1", tokenID)
		}(id)
	}

	store := user.NewStore(db)
	assert.NoError(t, store.ChangeOwnPassword(userID.String(), "old-password", "new-password", currentID.String()))

	updated, err := store.GetUserByID(userID.String())
	assert.NoError(t, err)
	assert.True(t, auth.ValidatePassword(updated.PasswordHashed, "new-password"))
	current, err := refreshStore.GetRefreshTokenByID(currentID.String())
	assert.NoError(t, err)
	assert.Nil(t, current.RevokedAt)
	other, err := refreshStore.GetRefreshTokenByID(otherID.String())
	assert.NoError(t, err)
	assert.NotNil(t, other.RevokedAt)
}

func TestChangeOwnPasswordRejectsGoogleManagedAccount(t *testing.T) {
	db := openTestDB(t)
	userID := uuid.New()
	passwordHash, err := auth.HashPassword("unusable-google-password")
	assert.NoError(t, err)
	insertUser(db, types.User{
		ID: userID, Username: "google-account-test", Firstname: "Google", Lastname: "Test",
		Email: "google-account-" + userID.String()[:8] + "@example.com", PasswordHashed: passwordHash,
		ExternalType: "google", ExternalID: "google-" + userID.String(),
		CreateTime: time.Now(), IsActive: true,
	})
	defer cleanUser(db, userID)

	err = user.NewStore(db).ChangeOwnPassword(userID.String(), "anything", "new-password", "")
	assert.ErrorIs(t, err, types.ErrPasswordChangeUnavailable)
}
