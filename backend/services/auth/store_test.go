package auth_test

import (
	"database/sql"
	"errors"
	"expense-tracker/backend/config"
	"expense-tracker/backend/db"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRefreshTokenConcurrentRotationCreatesOneValidSuccessor(t *testing.T) {
	database := openRefreshTestDB(t)
	userID := insertRefreshTestUser(t, database)
	store := auth.NewRefreshStore(database)
	familyID := uuid.New()
	predecessor := types.RefreshToken{
		ID:        uuid.New(),
		FamilyID:  familyID,
		UserID:    userID,
		TokenHash: "predecessor-hash-" + uuid.NewString(),
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
	require.NoError(t, store.CreateRefreshToken(predecessor))

	successors := []types.RefreshToken{
		newRefreshTestToken(userID, "successor-one-"+uuid.NewString()),
		newRefreshTestToken(userID, "successor-two-"+uuid.NewString()),
	}
	start := make(chan struct{})
	results := make(chan error, len(successors))
	var wg sync.WaitGroup
	for _, successor := range successors {
		wg.Go(func() {
			<-start
			results <- store.RotateRefreshToken(predecessor.ID.String(), predecessor.TokenHash, successor)
		})
	}
	close(start)
	wg.Wait()
	close(results)

	successCount := 0
	invalidCount := 0
	for err := range results {
		switch {
		case err == nil:
			successCount++
		case errors.Is(err, types.ErrInvalidToken):
			invalidCount++
		default:
			require.NoError(t, err)
		}
	}
	assert.Equal(t, 1, successCount)
	assert.Equal(t, 1, invalidCount)

	var activeCount int
	require.NoError(t, database.QueryRow(
		"SELECT COUNT(*) FROM refresh_tokens WHERE family_id = $1 AND revoked_at IS NULL",
		familyID,
	).Scan(&activeCount))
	assert.Equal(t, 1, activeCount)

	var totalCount int
	require.NoError(t, database.QueryRow(
		"SELECT COUNT(*) FROM refresh_tokens WHERE family_id = $1",
		familyID,
	).Scan(&totalCount))
	assert.Equal(t, 2, totalCount)
}

func TestRefreshTokenRotationRollsBackPredecessorWhenSuccessorInsertFails(t *testing.T) {
	database := openRefreshTestDB(t)
	userID := insertRefreshTestUser(t, database)
	store := auth.NewRefreshStore(database)
	predecessor := newRefreshTestToken(userID, "rollback-predecessor-"+uuid.NewString())
	blocker := newRefreshTestToken(userID, "rollback-blocker-"+uuid.NewString())
	require.NoError(t, store.CreateRefreshToken(predecessor))
	require.NoError(t, store.CreateRefreshToken(blocker))

	successor := newRefreshTestToken(userID, "rollback-successor-"+uuid.NewString())
	successor.ID = blocker.ID
	require.Error(t, store.RotateRefreshToken(predecessor.ID.String(), predecessor.TokenHash, successor))

	stored, err := store.GetRefreshTokenByID(predecessor.ID.String())
	require.NoError(t, err)
	assert.Nil(t, stored.RevokedAt)
}

func TestRefreshTokenReuseRevokesActiveDescendantOnlyInItsFamily(t *testing.T) {
	database := openRefreshTestDB(t)
	userID := insertRefreshTestUser(t, database)
	store := auth.NewRefreshStore(database)
	familyID := uuid.New()
	now := time.Now()
	predecessor := newRefreshTestToken(userID, "reused-predecessor-"+uuid.NewString())
	predecessor.FamilyID = familyID
	predecessor.RevokedAt = &now
	descendant := newRefreshTestToken(userID, "reused-descendant-"+uuid.NewString())
	descendant.FamilyID = familyID
	independent := newRefreshTestToken(userID, "reused-independent-"+uuid.NewString())
	for _, token := range []types.RefreshToken{predecessor, descendant, independent} {
		require.NoError(t, store.CreateRefreshToken(token))
	}

	candidate := newRefreshTestToken(userID, "reused-candidate-"+uuid.NewString())
	err := store.RotateRefreshToken(predecessor.ID.String(), predecessor.TokenHash, candidate)
	require.ErrorIs(t, err, types.ErrInvalidToken)

	storedDescendant, err := store.GetRefreshTokenByID(descendant.ID.String())
	require.NoError(t, err)
	assert.NotNil(t, storedDescendant.RevokedAt)
	storedIndependent, err := store.GetRefreshTokenByID(independent.ID.String())
	require.NoError(t, err)
	assert.Nil(t, storedIndependent.RevokedAt)
	_, err = store.GetRefreshTokenByID(candidate.ID.String())
	require.ErrorIs(t, err, types.ErrInvalidToken)
}

func TestRevokeRefreshTokenFamilyDoesNotRevokeIndependentSession(t *testing.T) {
	database := openRefreshTestDB(t)
	userID := insertRefreshTestUser(t, database)
	store := auth.NewRefreshStore(database)
	familyID := uuid.New()
	predecessor := newRefreshTestToken(userID, "family-predecessor-"+uuid.NewString())
	predecessor.FamilyID = familyID
	successor := newRefreshTestToken(userID, "family-successor-"+uuid.NewString())
	successor.FamilyID = familyID
	independent := newRefreshTestToken(userID, "independent-"+uuid.NewString())
	for _, token := range []types.RefreshToken{predecessor, successor, independent} {
		require.NoError(t, store.CreateRefreshToken(token))
	}

	require.NoError(t, store.RevokeRefreshTokenFamily(predecessor.ID.String()))

	for _, id := range []uuid.UUID{predecessor.ID, successor.ID} {
		stored, err := store.GetRefreshTokenByID(id.String())
		require.NoError(t, err)
		assert.NotNil(t, stored.RevokedAt)
	}
	storedIndependent, err := store.GetRefreshTokenByID(independent.ID.String())
	require.NoError(t, err)
	assert.Nil(t, storedIndependent.RevokedAt)
}

func openRefreshTestDB(t *testing.T) *sql.DB {
	t.Helper()

	database, err := db.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		t.Skipf("skipping: db connect error: %v", err)
	}
	if err := database.Ping(); err != nil {
		_ = database.Close()
		t.Skipf("skipping: db ping error: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})
	return database
}

func insertRefreshTestUser(t *testing.T, database *sql.DB) uuid.UUID {
	t.Helper()

	userID := uuid.New()
	_, err := database.Exec(
		`INSERT INTO users (
			id, username, firstname, lastname, nickname, email, password_hash,
			has_local_password, external_type, external_id, create_time_utc, is_active
		) VALUES ($1, $2, $3, $4, $5, $6, $7, TRUE, NULL, NULL, $8, TRUE)`,
		userID,
		"refresh-test-"+userID.String()[:8],
		"Refresh",
		"Test",
		"Refresh Test",
		"refresh-test-"+userID.String()+"@example.com",
		"test-only-password-hash",
		time.Now().UTC(),
	)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, _ = database.Exec("DELETE FROM users WHERE id = $1", userID)
	})
	return userID
}

func newRefreshTestToken(userID uuid.UUID, tokenHash string) types.RefreshToken {
	id := uuid.New()
	return types.RefreshToken{
		ID:        id,
		FamilyID:  id,
		UserID:    userID,
		TokenHash: tokenHash,
		ExpiresAt: time.Now().Add(time.Hour),
		CreatedAt: time.Now(),
	}
}
