package user

import (
	"bytes"
	"encoding/json"
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type accountStoreMock struct {
	user             *types.User
	updateProfileFn  func(string, types.UpdateOwnProfilePayload) error
	changePasswordFn func(string, string, string, string) error
}

func (m *accountStoreMock) GetUserByID(string) (*types.User, error) {
	return m.user, nil
}

func (m *accountStoreMock) UpdateOwnProfile(userID string, payload types.UpdateOwnProfilePayload) error {
	if m.updateProfileFn != nil {
		return m.updateProfileFn(userID, payload)
	}
	m.user.Firstname = payload.Firstname
	m.user.Lastname = payload.Lastname
	m.user.Nickname = payload.Nickname
	return nil
}

func (m *accountStoreMock) ChangeOwnPassword(userID string, currentPassword string, newPassword string, preserveRefreshID string) error {
	if m.changePasswordFn != nil {
		return m.changePasswordFn(userID, currentPassword, newPassword, preserveRefreshID)
	}
	return nil
}

func accountTestRequest(t *testing.T, method string, path string, body string, userID uuid.UUID) *http.Request {
	t.Helper()
	token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), userID)
	assert.NoError(t, err)
	request := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	return request
}

func accountTestRouter(store AccountStore) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	NewProtectedHandler(store).RegisterRoutes(router.Group(""))
	return router
}

func TestAccountReturnsSafeCapabilityData(t *testing.T) {
	userID := uuid.New()
	store := &accountStoreMock{user: &types.User{
		ID: userID, Firstname: "Local", Lastname: "User", Email: "local@example.com",
		PasswordHashed: "must-not-leak", IsActive: true,
	}}
	response := httptest.NewRecorder()
	accountTestRouter(store).ServeHTTP(response, accountTestRequest(t, http.MethodGet, "/account", "", userID))

	assert.Equal(t, http.StatusOK, response.Code)
	assert.NotContains(t, response.Body.String(), "must-not-leak")
	assert.Contains(t, response.Body.String(), `"passwordChangeAllowed":true`)
	assert.Contains(t, response.Body.String(), `"googleConnected":false`)
}

func TestAccountRequiresAuthentication(t *testing.T) {
	store := &accountStoreMock{user: &types.User{IsActive: true}}
	response := httptest.NewRecorder()
	accountTestRouter(store).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/account", nil))

	assert.Equal(t, http.StatusUnauthorized, response.Code)
}

func TestUpdateProfileUsesAuthenticatedUserAndKeepsEmailImmutable(t *testing.T) {
	userID := uuid.New()
	called := false
	store := &accountStoreMock{
		user: &types.User{ID: userID, Email: "original@example.com", IsActive: true},
		updateProfileFn: func(actualID string, payload types.UpdateOwnProfilePayload) error {
			called = true
			assert.Equal(t, userID.String(), actualID)
			assert.Equal(t, "Taylor", payload.Firstname)
			assert.Equal(t, "Swift", payload.Lastname)
			assert.Equal(t, "T", payload.Nickname)
			return nil
		},
	}
	response := httptest.NewRecorder()
	request := accountTestRequest(t, http.MethodPatch, "/account", `{"firstname":" Taylor ","lastname":" Swift ","nickname":" T ","email":"attacker@example.com"}`, userID)
	accountTestRouter(store).ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.True(t, called)
	var payload types.AccountResponse
	assert.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	assert.Equal(t, "original@example.com", payload.Email)
}

func TestChangePasswordPassesCurrentRefreshSession(t *testing.T) {
	userID := uuid.New()
	refreshToken, refreshID, _, err := auth.CreateRefreshJWT([]byte(config.Envs.RefreshJWTSecret), userID)
	assert.NoError(t, err)
	called := false
	store := &accountStoreMock{
		user: &types.User{ID: userID, IsActive: true},
		changePasswordFn: func(actualID string, current string, next string, preserve string) error {
			called = true
			assert.Equal(t, userID.String(), actualID)
			assert.Equal(t, "old-password", current)
			assert.Equal(t, "new-password", next)
			assert.Equal(t, refreshID, preserve)
			return nil
		},
	}
	response := httptest.NewRecorder()
	request := accountTestRequest(t, http.MethodPatch, "/account/password", `{"currentPassword":"old-password","newPassword":"new-password"}`, userID)
	request.AddCookie(&http.Cookie{Name: "refresh_token", Value: refreshToken})
	accountTestRouter(store).ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.True(t, called)
}

func TestChangePasswordRejectsGoogleManagedAccount(t *testing.T) {
	userID := uuid.New()
	store := &accountStoreMock{
		user: &types.User{ID: userID, IsActive: true, ExternalType: "google"},
		changePasswordFn: func(string, string, string, string) error {
			return types.ErrPasswordChangeUnavailable
		},
	}
	response := httptest.NewRecorder()
	request := accountTestRequest(t, http.MethodPatch, "/account/password", `{"currentPassword":"old-password","newPassword":"new-password"}`, userID)
	accountTestRouter(store).ServeHTTP(response, request)

	assert.Equal(t, http.StatusConflict, response.Code)
	assert.Contains(t, response.Body.String(), "password_change_unavailable")
}

func TestChangePasswordReportsIncorrectCurrentPassword(t *testing.T) {
	userID := uuid.New()
	store := &accountStoreMock{
		user: &types.User{ID: userID, IsActive: true},
		changePasswordFn: func(string, string, string, string) error {
			return types.ErrCurrentPasswordIncorrect
		},
	}
	response := httptest.NewRecorder()
	request := accountTestRequest(t, http.MethodPatch, "/account/password", `{"currentPassword":"wrong-password","newPassword":"new-password"}`, userID)
	accountTestRouter(store).ServeHTTP(response, request)

	assert.Equal(t, http.StatusUnauthorized, response.Code)
	assert.Contains(t, response.Body.String(), "current_password_incorrect")
}
