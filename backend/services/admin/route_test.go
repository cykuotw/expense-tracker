package admin

import (
	"bytes"
	"encoding/json"
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type userStoreMock struct {
	getUsersFn  func() ([]types.AdminUserResponse, error)
	setActiveFn func(string, string, bool) error
	setRoleFn   func(string, string, string) error
}

func (m *userStoreMock) GetAdminUsers() ([]types.AdminUserResponse, error) {
	return m.getUsersFn()
}
func (m *userStoreMock) SetUserActive(actorID string, targetID string, active bool) error {
	return m.setActiveFn(actorID, targetID, active)
}
func (m *userStoreMock) SetUserRole(actorID string, targetID string, role string) error {
	return m.setRoleFn(actorID, targetID, role)
}

type invitationStoreMock struct {
	getFn    func() ([]types.AdminInvitationResponse, error)
	tokenFn  func(string) (string, error)
	expireFn func(string) error
}

func (m *invitationStoreMock) GetAdminInvitations() ([]types.AdminInvitationResponse, error) {
	return m.getFn()
}
func (m *invitationStoreMock) RotateInvitationTokenByID(id string) (string, error) {
	return m.tokenFn(id)
}
func (m *invitationStoreMock) ExpireInvitationByID(id string) error {
	return m.expireFn(id)
}

func adminTestRouter(users UserStore, invitations InvitationStore) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	NewHandler(users, invitations).RegisterRoutes(router.Group(""))
	return router
}

func authenticatedRequest(t *testing.T, method string, path string, body []byte, actor uuid.UUID) *http.Request {
	t.Helper()
	token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), actor)
	assert.NoError(t, err)
	request := httptest.NewRequest(method, path, bytes.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	return request
}

func baseInvitationMock() *invitationStoreMock {
	return &invitationStoreMock{
		getFn:    func() ([]types.AdminInvitationResponse, error) { return []types.AdminInvitationResponse{}, nil },
		tokenFn:  func(string) (string, error) { return "token", nil },
		expireFn: func(string) error { return nil },
	}
}

func TestListReturnsSafeManagementData(t *testing.T) {
	userID := uuid.New()
	invitationID := uuid.New()
	users := &userStoreMock{
		getUsersFn: func() ([]types.AdminUserResponse, error) {
			return []types.AdminUserResponse{{ID: userID, Email: "user@example.com", IsActive: true, Role: "admin", IsProtectedAdmin: true}}, nil
		},
		setActiveFn: func(string, string, bool) error { return nil },
		setRoleFn:   func(string, string, string) error { return nil },
	}
	invitations := baseInvitationMock()
	invitations.getFn = func() ([]types.AdminInvitationResponse, error) {
		return []types.AdminInvitationResponse{{ID: invitationID, Email: "invite@example.com", Status: "invited", ExpiresAt: time.Now().Add(time.Hour)}}, nil
	}

	response := httptest.NewRecorder()
	adminTestRouter(users, invitations).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/users", nil))

	assert.Equal(t, http.StatusOK, response.Code)
	assert.NotContains(t, response.Body.String(), "token")
	assert.Contains(t, response.Body.String(), "invite@example.com")
	assert.Contains(t, response.Body.String(), `"isProtectedAdmin":true`)
}

func TestListReturnsEmptyArraysForNilStoreResults(t *testing.T) {
	users := &userStoreMock{
		getUsersFn:  func() ([]types.AdminUserResponse, error) { return nil, nil },
		setActiveFn: func(string, string, bool) error { return nil },
		setRoleFn:   func(string, string, string) error { return nil },
	}
	invitations := baseInvitationMock()
	invitations.getFn = func() ([]types.AdminInvitationResponse, error) { return nil, nil }

	response := httptest.NewRecorder()
	adminTestRouter(users, invitations).ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/admin/users", nil))

	assert.Equal(t, http.StatusOK, response.Code)
	assert.JSONEq(t, `{"users":[],"invitations":[]}`, response.Body.String())
}

func TestUpdateStatusRejectsProtectedChange(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	users := &userStoreMock{
		getUsersFn: func() ([]types.AdminUserResponse, error) { return nil, nil },
		setActiveFn: func(string, string, bool) error {
			return types.ErrProtectedAdmin
		},
		setRoleFn: func(string, string, string) error { return nil },
	}

	response := httptest.NewRecorder()
	request := authenticatedRequest(t, http.MethodPatch, "/admin/users/"+target.String()+"/status", []byte(`{"isActive":true}`), actor)
	adminTestRouter(users, baseInvitationMock()).ServeHTTP(response, request)

	assert.Equal(t, http.StatusConflict, response.Code)
	var payload struct {
		Code string `json:"code"`
	}
	assert.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	assert.Equal(t, "PROTECTED_ADMIN", payload.Code)
}

func TestUpdateStatusPassesActorAndTarget(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	called := false
	users := &userStoreMock{
		getUsersFn: func() ([]types.AdminUserResponse, error) { return nil, nil },
		setActiveFn: func(actorID string, targetID string, active bool) error {
			called = true
			assert.Equal(t, actor.String(), actorID)
			assert.Equal(t, target.String(), targetID)
			assert.False(t, active)
			return nil
		},
		setRoleFn: func(string, string, string) error { return nil },
	}

	response := httptest.NewRecorder()
	request := authenticatedRequest(t, http.MethodPatch, "/admin/users/"+target.String()+"/status", []byte(`{"isActive":false}`), actor)
	adminTestRouter(users, baseInvitationMock()).ServeHTTP(response, request)

	assert.Equal(t, http.StatusOK, response.Code)
	assert.True(t, called)
}

func TestUpdateRoleRejectsProtectedChange(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	users := &userStoreMock{
		getUsersFn:  func() ([]types.AdminUserResponse, error) { return nil, nil },
		setActiveFn: func(string, string, bool) error { return nil },
		setRoleFn: func(string, string, string) error {
			return types.ErrProtectedAdmin
		},
	}

	response := httptest.NewRecorder()
	request := authenticatedRequest(t, http.MethodPatch, "/admin/users/"+target.String()+"/role", []byte(`{"role":"user"}`), actor)
	adminTestRouter(users, baseInvitationMock()).ServeHTTP(response, request)

	assert.Equal(t, http.StatusConflict, response.Code)
	var payload struct {
		Code string `json:"code"`
	}
	assert.NoError(t, json.Unmarshal(response.Body.Bytes(), &payload))
	assert.Equal(t, "PROTECTED_ADMIN", payload.Code)
	assert.NotContains(t, response.Body.String(), "admin@example.com")
}

func TestUpdateRoleRejectsUnknownRole(t *testing.T) {
	actor := uuid.New()
	target := uuid.New()
	users := &userStoreMock{
		getUsersFn:  func() ([]types.AdminUserResponse, error) { return nil, nil },
		setActiveFn: func(string, string, bool) error { return nil },
		setRoleFn: func(string, string, string) error {
			t.Fatal("store must not receive an invalid role")
			return nil
		},
	}

	response := httptest.NewRecorder()
	request := authenticatedRequest(t, http.MethodPatch, "/admin/users/"+target.String()+"/role", []byte(`{"role":"owner"}`), actor)
	adminTestRouter(users, baseInvitationMock()).ServeHTTP(response, request)

	assert.Equal(t, http.StatusBadRequest, response.Code)
}
