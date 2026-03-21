package api

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-tracker/backend/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewBaseRouterRecoversFromPanics(t *testing.T) {
	originalMode := config.Envs.Mode
	config.Envs.Mode = "release"
	t.Cleanup(func() {
		config.Envs.Mode = originalMode
	})

	gin.SetMode(gin.ReleaseMode)
	router := newBaseRouter()
	router.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestNewHTTPServerAppliesTimeouts(t *testing.T) {
	handler := http.NewServeMux()
	server := newHTTPServer("127.0.0.1:8000", handler)

	assert.Equal(t, "127.0.0.1:8000", server.Addr)
	assert.Same(t, handler, server.Handler)
	assert.Equal(t, readHeaderTimeout, server.ReadHeaderTimeout)
	assert.Equal(t, readTimeout, server.ReadTimeout)
	assert.Equal(t, writeTimeout, server.WriteTimeout)
	assert.Equal(t, idleTimeout, server.IdleTimeout)
}
