package tracker

import (
	"database/sql"
	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/services/middleware"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type authorizationFixture struct {
	aliceID    uuid.UUID
	bobID      uuid.UUID
	aliceGroup uuid.UUID
	bobGroup   uuid.UUID
	bobBalance uuid.UUID
}

func TestCrossResourceAuthorizationDoesNotMutateData(t *testing.T) {
	db := openAuthorizationTestDB(t)
	fixture := createAuthorizationFixture(t, db)
	handler := NewHandler(db)

	t.Run("cannot archive another creator's group", func(t *testing.T) {
		response := serveAuthenticatedRequest(t, handler, http.MethodPut,
			config.Envs.APIPath+"/archive_group/"+fixture.bobGroup.String(), fixture.aliceID)
		require.Equal(t, http.StatusNotFound, response.Code)

		var active bool
		require.NoError(t, db.QueryRow("SELECT is_active FROM groups WHERE id = $1", fixture.bobGroup).Scan(&active))
		require.True(t, active)
	})

	t.Run("cannot settle a balance through another group", func(t *testing.T) {
		response := serveAuthenticatedRequest(t, handler, http.MethodPost,
			config.Envs.APIPath+"/settle_balance/"+fixture.aliceGroup.String()+"/"+fixture.bobBalance.String(), fixture.aliceID)
		require.Equal(t, http.StatusNotFound, response.Code)

		var settled bool
		require.NoError(t, db.QueryRow("SELECT is_settled FROM balance WHERE id = $1", fixture.bobBalance).Scan(&settled))
		require.False(t, settled)
	})
}

func openAuthorizationTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := dbstore.NewPostgreSQLStorage(config.Envs)
	if err != nil {
		t.Skipf("skipping PostgreSQL authorization integration test: %v", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		t.Skipf("skipping PostgreSQL authorization integration test: %v", err)
	}
	t.Cleanup(func() { require.NoError(t, db.Close()) })
	return db
}

func createAuthorizationFixture(t *testing.T, db *sql.DB) authorizationFixture {
	t.Helper()
	fixture := authorizationFixture{
		aliceID:    uuid.New(),
		bobID:      uuid.New(),
		aliceGroup: uuid.New(),
		bobGroup:   uuid.New(),
		bobBalance: uuid.New(),
	}
	now := time.Now().UTC()

	for _, user := range []struct {
		id       uuid.UUID
		username string
	}{
		{id: fixture.aliceID, username: "authorization-alice-" + fixture.aliceID.String()},
		{id: fixture.bobID, username: "authorization-bob-" + fixture.bobID.String()},
	} {
		_, err := db.Exec(`INSERT INTO users (id, username, firstname, lastname, email, password_hash, create_time_utc, is_active, has_local_password, role)
			VALUES ($1, $2, 'Authorization', 'Test', $3, 'not-used', $4, TRUE, TRUE, 'user')`,
			user.id, user.username, user.username+"@example.test", now)
		require.NoError(t, err)
	}

	for _, group := range []struct {
		id      uuid.UUID
		creator uuid.UUID
		name    string
	}{
		{id: fixture.aliceGroup, creator: fixture.aliceID, name: "authorization-alice"},
		{id: fixture.bobGroup, creator: fixture.bobID, name: "authorization-bob"},
	} {
		_, err := db.Exec(`INSERT INTO groups (id, group_name, description, create_time_utc, is_active, create_by_user_id, currency)
			VALUES ($1, $2, '', $3, TRUE, $4, 'CAD')`, group.id, group.name, now, group.creator)
		require.NoError(t, err)
		_, err = db.Exec(`INSERT INTO group_member (id, group_id, user_id) VALUES ($1, $2, $3)`, uuid.New(), group.id, group.creator)
		require.NoError(t, err)
	}

	_, err := db.Exec(`INSERT INTO balance (id, sender_user_id, receiver_user_id, share, group_id, create_time_utc, is_outdated, is_settled)
		VALUES ($1, $2, $2, 10, $3, $4, FALSE, FALSE)`, fixture.bobBalance, fixture.bobID, fixture.bobGroup, now)
	require.NoError(t, err)

	t.Cleanup(func() {
		_, err := db.Exec("DELETE FROM groups WHERE id = ANY($1)", []uuid.UUID{fixture.aliceGroup, fixture.bobGroup})
		require.NoError(t, err)
		_, err = db.Exec("DELETE FROM users WHERE id = ANY($1)", []uuid.UUID{fixture.aliceID, fixture.bobID})
		require.NoError(t, err)
	})

	return fixture
}

func serveAuthenticatedRequest(t *testing.T, handler http.Handler, method string, path string, actorID uuid.UUID) *httptest.ResponseRecorder {
	t.Helper()
	token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), actorID)
	require.NoError(t, err)

	request := httptest.NewRequest(method, path, nil)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Origin", config.Envs.FrontendOrigin)
	request.Header.Set(middleware.CSRFHeaderName, "authorization-test-csrf")
	request.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: "authorization-test-csrf"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
