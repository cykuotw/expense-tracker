package middleware

import (
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

type adminUserStoreMock struct {
	user *types.User
}

func (m adminUserStoreMock) GetUserByID(string) (*types.User, error) {
	return m.user, nil
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
