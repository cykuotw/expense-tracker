package agenttools

import (
	"net/http/httptest"
	"testing"
	"time"

	"expense-tracker/backend/config"
	"expense-tracker/internal/agentcontract"
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/require"
)

func TestLocalToolsRetainJWTAndDomainChecks(t *testing.T) {
	t.Setenv("AGENT_MODE", "local")
	f := &fixture{active: true, member: true}
	handler, err := NewLocalHandler(f, f, f)
	require.NoError(t, err)
	router := gin.New()
	handler.RegisterRoutes(router.Group(""))
	valid := token(t, testActor.String(), "access", config.Envs.JWTSecret, jwt.NewNumericDate(time.Now().Add(time.Hour)))
	call := func(path, access, peer string) int {
		request := httptest.NewRequest("GET", "http://127.0.0.1:8000"+path, nil)
		request.RemoteAddr = peer
		request.Header.Set(agentcontract.AccessTokenHeader, access)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		return response.Code
	}
	require.Equal(t, 200, call(agentcontract.PreflightPath, valid, "127.0.0.1:9000"))
	require.Equal(t, 401, call(agentcontract.PreflightPath, "", "127.0.0.1:9000"))
	require.Equal(t, 401, call(agentcontract.PreflightPath, "invalid", "127.0.0.1:9000"))
	expired := token(t, testActor.String(), "access", config.Envs.JWTSecret, jwt.NewNumericDate(time.Now().Add(-time.Hour)))
	require.Equal(t, 401, call(agentcontract.PreflightPath, expired, "127.0.0.1:9000"))
	require.Equal(t, 403, call(agentcontract.PreflightPath, valid, "192.0.2.1:9000"))
	f.active = false
	require.Equal(t, 403, call(agentcontract.PreflightPath, valid, "127.0.0.1:9000"))
	f.active = true
	f.member = false
	require.Equal(t, 404, call(agentcontract.Prefix+"/balance/"+testGroup.String(), valid, "127.0.0.1:9000"))

	// Environment changes alone cannot weaken the production/default handler.
	request := httptest.NewRequest("GET", "http://127.0.0.1:8000"+agentcontract.PreflightPath, nil)
	request.RemoteAddr = "127.0.0.1:9000"
	request.Header.Set(agentcontract.AccessTokenHeader, valid)
	response := httptest.NewRecorder()
	fixtureRouter(f, testRole).ServeHTTP(response, request)
	require.Equal(t, 403, response.Code)
}

func TestLocalToolsRejectLambdaAndRelease(t *testing.T) {
	f := new(fixture)
	t.Run("Lambda", func(t *testing.T) {
		t.Setenv("AWS_LAMBDA_RUNTIME_API", "synthetic")
		_, err := NewLocalHandler(f, f, f)
		require.Error(t, err)
	})
	original := config.BuildMode
	config.BuildMode = "release"
	t.Cleanup(func() { config.BuildMode = original })
	_, err := NewLocalHandler(f, f, f)
	require.Error(t, err)
}
