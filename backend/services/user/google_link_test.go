package user

import (
	"context"
	googleAuth "expense-tracker/backend/services/auth/google"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type googleVerifierStub struct {
	claims *types.VerifiedGoogleClaims
	err    error
}

func (v *googleVerifierStub) VerifyGoogleIDToken(context.Context, string) (*types.VerifiedGoogleClaims, error) {
	return v.claims, v.err
}

func verifiedGoogleClaims() *types.VerifiedGoogleClaims {
	verified := true
	return &types.VerifiedGoogleClaims{
		Subject:       "google-sub-123",
		Email:         " LOCAL@example.com ",
		EmailVerified: &verified,
	}
}

func googleLinkRouter(handler gin.HandlerFunc) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.POST("/account/google/link", handler)
	return router
}

func TestGoogleLinkInProcessUsesAuthenticatedAccountAndVerifiedClaims(t *testing.T) {
	userID := uuid.New()
	store := &accountStoreMock{user: &types.User{
		ID: userID, Email: "local@example.com", HasLocalPassword: true, IsActive: true,
	}}
	store.linkGoogleFn = func(_ context.Context, actualUserID string, password string, externalID string, email string, preserve string) error {
		assert.Equal(t, userID.String(), actualUserID)
		assert.Equal(t, "current-password", password)
		assert.Equal(t, "google-sub-123", externalID)
		assert.Equal(t, "local@example.com", email)
		assert.Empty(t, preserve)
		store.user.ExternalType = "google"
		store.user.ExternalID = externalID
		return nil
	}
	handler := NewProtectedHandler(store)
	handler.googleVerifier = &googleVerifierStub{claims: verifiedGoogleClaims()}

	request := accountTestRequest(t, http.MethodPost, "/account/google/link", `{"currentPassword":"current-password"}`, userID)
	request.Header.Set("Authorization", "Bearer google-id-token")
	response := httptest.NewRecorder()
	googleLinkRouter(handler.handleGoogleLinkInProcess).ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.Contains(t, response.Body.String(), `"googleConnected":true`)
	assert.Contains(t, response.Body.String(), `"passwordChangeAllowed":true`)
}

func TestGoogleLinkUpstreamUsesAuthorizerClaims(t *testing.T) {
	userID := uuid.New()
	called := false
	store := &accountStoreMock{
		user: &types.User{ID: userID, Email: "local@example.com", HasLocalPassword: true, IsActive: true},
		linkGoogleFn: func(_ context.Context, _ string, _ string, _ string, _ string, _ string) error {
			called = true
			return nil
		},
	}
	handler := NewProtectedHandler(store)
	request := accountTestRequest(t, http.MethodPost, "/account/google/link", `{"currentPassword":"current-password"}`, userID)
	request = request.WithContext(googleAuth.ContextWithVerifiedClaims(request.Context(), verifiedGoogleClaims()))
	response := httptest.NewRecorder()
	googleLinkRouter(handler.handleGoogleLinkUpstreamVerified).ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.True(t, called)
}

func TestGoogleLinkReturnsStableConflictErrors(t *testing.T) {
	tests := []struct {
		name string
		err  error
		code string
	}{
		{name: "email mismatch", err: types.ErrGoogleLinkEmailMismatch, code: "google_link_email_mismatch"},
		{name: "identity in use", err: types.ErrGoogleAccountConflict, code: "google_account_conflict"},
		{name: "already connected", err: types.ErrGoogleAlreadyConnected, code: "google_already_connected"},
		{name: "link unavailable", err: types.ErrGoogleLinkUnavailable, code: "google_link_unavailable"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			userID := uuid.New()
			store := &accountStoreMock{
				user:         &types.User{ID: userID, Email: "local@example.com", HasLocalPassword: true, IsActive: true},
				linkGoogleFn: func(context.Context, string, string, string, string, string) error { return test.err },
			}
			handler := NewProtectedHandler(store)
			handler.googleVerifier = &googleVerifierStub{claims: verifiedGoogleClaims()}
			request := accountTestRequest(t, http.MethodPost, "/account/google/link", `{"currentPassword":"current-password"}`, userID)
			request.Header.Set("Authorization", "Bearer google-id-token")
			response := httptest.NewRecorder()
			googleLinkRouter(handler.handleGoogleLinkInProcess).ServeHTTP(response, request)

			assert.Equal(t, http.StatusConflict, response.Code)
			assert.Contains(t, response.Body.String(), `"code":"`+test.code+`"`)
		})
	}
}

func TestGoogleLinkRejectsIncorrectCurrentPassword(t *testing.T) {
	userID := uuid.New()
	store := &accountStoreMock{
		user: &types.User{ID: userID, Email: "local@example.com", HasLocalPassword: true, IsActive: true},
		linkGoogleFn: func(context.Context, string, string, string, string, string) error {
			return types.ErrCurrentPasswordIncorrect
		},
	}
	handler := NewProtectedHandler(store)
	handler.googleVerifier = &googleVerifierStub{claims: verifiedGoogleClaims()}
	request := accountTestRequest(t, http.MethodPost, "/account/google/link", `{"currentPassword":"wrong-password"}`, userID)
	request.Header.Set("Authorization", "Bearer google-id-token")
	response := httptest.NewRecorder()
	googleLinkRouter(handler.handleGoogleLinkInProcess).ServeHTTP(response, request)

	assert.Equal(t, http.StatusUnauthorized, response.Code)
	assert.Contains(t, response.Body.String(), `"code":"current_password_incorrect"`)
}
