package tracker

import (
	"bytes"
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/services/middleware"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestReceiptOCRGrantRouteRequiresAdministrator(t *testing.T) {
	database := openAuthorizationTestDB(t)
	adminID := uuid.New()
	regularID := uuid.New()
	targetID := uuid.New()
	for _, fixture := range []struct {
		id   uuid.UUID
		role string
	}{
		{id: adminID, role: "admin"},
		{id: regularID, role: "user"},
		{id: targetID, role: "user"},
	} {
		_, err := database.ExecContext(t.Context(), `
			INSERT INTO users (
				id, username, firstname, lastname, email, password_hash,
				create_time_utc, is_active, has_local_password, role
			) VALUES ($1, $2, 'Grant', 'Authorization', $3, 'not-used', $4, TRUE, TRUE, $5)`,
			fixture.id,
			"grant-authorization-"+fixture.id.String()[:8],
			"grant-authorization-"+fixture.id.String()[:8]+"@example.test",
			time.Now().UTC(),
			fixture.role,
		)
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		_, err := database.Exec(
			"DELETE FROM user_feature_grant WHERE user_id = ANY($1) OR granted_by_user_id = ANY($1)",
			[]uuid.UUID{adminID, regularID, targetID},
		)
		require.NoError(t, err)
		_, err = database.Exec(
			"DELETE FROM users WHERE id = ANY($1)", []uuid.UUID{adminID, regularID, targetID})
		require.NoError(t, err)
	})

	handler := NewHandler(database)
	path := config.Envs.APIPath + "/admin/users/" + targetID.String() + "/features/receipt-ocr"

	response := serveGrantRequest(t, handler, path, regularID, true)
	require.Equal(t, http.StatusForbidden, response.Code)
	var grantCount int
	require.NoError(t, database.QueryRowContext(t.Context(), `
		SELECT COUNT(*) FROM user_feature_grant
		WHERE user_id = $1 AND feature_key = 'receipt_ocr'`, targetID).Scan(&grantCount))
	require.Zero(t, grantCount)

	response = serveGrantRequest(t, handler, path, adminID, true)
	require.Equal(t, http.StatusOK, response.Code)
	require.NoError(t, database.QueryRowContext(t.Context(), `
		SELECT COUNT(*) FROM user_feature_grant
		WHERE user_id = $1 AND feature_key = 'receipt_ocr'`, targetID).Scan(&grantCount))
	require.Equal(t, 1, grantCount)
}

func serveGrantRequest(t *testing.T, handler http.Handler, path string, actorID uuid.UUID, enabled bool) *httptest.ResponseRecorder {
	t.Helper()
	token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), actorID)
	require.NoError(t, err)
	body := []byte(`{"enabled":false}`)
	if enabled {
		body = []byte(`{"enabled":true}`)
	}
	request := httptest.NewRequest(http.MethodPatch, path, bytes.NewReader(body))
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", config.Envs.FrontendOrigin)
	request.Header.Set(middleware.CSRFHeaderName, "feature-grant-test-csrf")
	request.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: "feature-grant-test-csrf"})
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}
