package middleware

import (
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

type adminUserStoreMock struct {
	user *types.User
	err  error
}

func (m adminUserStoreMock) GetUserByID(string) (*types.User, error) {
	return m.user, m.err
}

func TestDatabaseFailureDoesNotBecomeAuthRejection(t *testing.T) {
	actor := uuid.New()
	token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), actor)
	if err != nil {
		t.Fatal(err)
	}
	for _, guard := range []gin.HandlerFunc{
		ActiveUserMiddleware(adminUserStoreMock{err: io.EOF}),
		AdminMiddleware(adminUserStoreMock{err: io.EOF}),
	} {
		router := gin.New()
		router.GET("/protected", guard, func(c *gin.Context) { t.Error("handler reached after database failure") })
		request := httptest.NewRequest("GET", "/protected", nil)
		request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		assert.Equal(t, http.StatusServiceUnavailable, response.Code)
		assert.Contains(t, response.Body.String(), "service_unavailable")
	}
}

func TestAdminMiddlewareAuthorization(t *testing.T) {
	for _, test := range []struct {
		name   string
		user   *types.User
		status int
	}{
		{name: "active admin", user: &types.User{Role: "admin", IsActive: true}, status: http.StatusNoContent},
		{name: "regular user", user: &types.User{Role: "user", IsActive: true}, status: http.StatusForbidden},
		{name: "inactive admin", user: &types.User{Role: "admin", IsActive: false}, status: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			actor := uuid.New()
			token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), actor)
			assert.NoError(t, err)

			gin.SetMode(gin.ReleaseMode)
			router := gin.New()
			router.GET("/admin", AdminMiddleware(adminUserStoreMock{user: test.user}), func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/admin", nil)
			request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			assert.Equal(t, test.status, response.Code)
		})
	}
}

func TestActiveUserMiddlewareRejectsInactiveAccounts(t *testing.T) {
	for _, test := range []struct {
		name   string
		active bool
		status int
	}{
		{name: "active user", active: true, status: http.StatusNoContent},
		{name: "inactive user", active: false, status: http.StatusForbidden},
	} {
		t.Run(test.name, func(t *testing.T) {
			actor := uuid.New()
			token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), actor)
			assert.NoError(t, err)

			gin.SetMode(gin.ReleaseMode)
			router := gin.New()
			router.GET("/protected", ActiveUserMiddleware(adminUserStoreMock{user: &types.User{IsActive: test.active}}), func(c *gin.Context) {
				c.Status(http.StatusNoContent)
			})
			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
			response := httptest.NewRecorder()

			router.ServeHTTP(response, request)

			assert.Equal(t, test.status, response.Code)
		})
	}
}
