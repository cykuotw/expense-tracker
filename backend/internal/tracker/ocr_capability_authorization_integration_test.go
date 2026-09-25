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

func TestOCRCapabilityMintUsesProtectedAccountBoundary(t *testing.T) {
	database := openAuthorizationTestDB(t)
	_, err := database.Exec("DELETE FROM users WHERE username LIKE 'ocr-boundary-%'")
	require.NoError(t, err)
	grantedID := uuid.New()
	ungrantedID := uuid.New()
	inactiveID := uuid.New()
	now := time.Now().UTC()
	for _, fixture := range []struct {
		id     uuid.UUID
		active bool
	}{
		{id: grantedID, active: true},
		{id: ungrantedID, active: true},
		{id: inactiveID, active: false},
	} {
		_, err := database.ExecContext(t.Context(), `
			INSERT INTO users (
				id, username, firstname, lastname, email, password_hash,
				create_time_utc, is_active, has_local_password, role
			) VALUES ($1, $2, 'OCR', 'Boundary', $3, 'not-used', $4, $5, TRUE, 'user')`,
			fixture.id,
			"ocr-boundary-"+fixture.id.String()[:8],
			"ocr-boundary-"+fixture.id.String()[:8]+"@example.test",
			now,
			fixture.active,
		)
		require.NoError(t, err)
	}
	t.Cleanup(func() {
		ids := []uuid.UUID{grantedID, ungrantedID, inactiveID}
		_, cleanupErr := database.Exec(
			"DELETE FROM user_feature_grant WHERE user_id = ANY($1) OR granted_by_user_id = ANY($1)", ids)
		require.NoError(t, cleanupErr)
		_, cleanupErr = database.Exec("DELETE FROM users WHERE id = ANY($1)", ids)
		require.NoError(t, cleanupErr)
	})
	_, err = database.ExecContext(t.Context(), `
		INSERT INTO user_feature_grant (
			user_id, feature_key, granted_by_user_id, granted_at
		) VALUES ($1, 'receipt_ocr', $1, $2), ($3, 'receipt_ocr', $3, $2)`,
		grantedID, now, inactiveID)
	require.NoError(t, err)
	handler := NewHandler(database)
	path := config.Envs.APIPath + "/ocr/capabilities"
	requestID := uuid.NewString()
	body := `{"requestId":"` + requestID + `","contentType":"image/jpeg"}`

	tests := []struct {
		name       string
		userID     *uuid.UUID
		origin     string
		csrfHeader string
		csrfCookie string
		status     int
	}{
		{name: "unauthenticated", origin: config.Envs.FrontendOrigin, csrfHeader: "csrf", csrfCookie: "csrf", status: http.StatusUnauthorized},
		{name: "wrong origin", userID: &grantedID, origin: "https://attacker.example.com", csrfHeader: "csrf", csrfCookie: "csrf", status: http.StatusForbidden},
		{name: "invalid csrf", userID: &grantedID, origin: config.Envs.FrontendOrigin, csrfHeader: "header", csrfCookie: "cookie", status: http.StatusForbidden},
		{name: "inactive", userID: &inactiveID, origin: config.Envs.FrontendOrigin, csrfHeader: "csrf", csrfCookie: "csrf", status: http.StatusForbidden},
		{name: "ungranted", userID: &ungrantedID, origin: config.Envs.FrontendOrigin, csrfHeader: "csrf", csrfCookie: "csrf", status: http.StatusForbidden},
		{name: "granted", userID: &grantedID, origin: config.Envs.FrontendOrigin, csrfHeader: "csrf", csrfCookie: "csrf", status: http.StatusCreated},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
			request.Header.Set("Content-Type", "application/json")
			request.Header.Set("Origin", test.origin)
			request.Header.Set(middleware.CSRFHeaderName, test.csrfHeader)
			request.AddCookie(&http.Cookie{Name: middleware.CSRFCookieName, Value: test.csrfCookie})
			if test.userID != nil {
				token, tokenErr := auth.CreateJWT([]byte(config.Envs.JWTSecret), *test.userID)
				require.NoError(t, tokenErr)
				request.Header.Set("Authorization", "Bearer "+token)
			}
			response := httptest.NewRecorder()
			handler.ServeHTTP(response, request)
			require.Equal(t, test.status, response.Code, response.Body.String())
		})
	}
}
