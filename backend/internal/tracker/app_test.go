package tracker

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-tracker/backend/config"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewHandlerRecoversFromPanics(t *testing.T) {
	originalMode := config.Envs.Mode
	config.Envs.Mode = "release"
	t.Cleanup(func() {
		config.Envs.Mode = originalMode
	})

	gin.SetMode(gin.ReleaseMode)
	router := NewHandler(nil)

	engine, ok := router.(*gin.Engine)
	if !ok {
		t.Fatalf("expected *gin.Engine, got %T", router)
	}

	engine.GET("/panic", func(c *gin.Context) {
		panic("boom")
	})

	req := httptest.NewRequest(http.MethodGet, "/panic", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusInternalServerError, rr.Code)
}

func TestNewHandlerRegistersHealthRoute(t *testing.T) {
	originalMode := config.Envs.Mode
	originalAPIPath := config.Envs.APIPath
	config.Envs.Mode = "release"
	config.Envs.APIPath = "/api/v0"
	t.Cleanup(func() {
		config.Envs.Mode = originalMode
		config.Envs.APIPath = originalAPIPath
	})

	router := NewHandler(nil)

	req := httptest.NewRequest(http.MethodGet, "/api/v0/health", nil)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.JSONEq(t, `{"status":"ok"}`, rr.Body.String())
}

func TestNewHTTPServerAppliesTimeouts(t *testing.T) {
	handler := http.NewServeMux()
	server := NewHTTPServer("127.0.0.1:8000", handler)

	assert.Equal(t, "127.0.0.1:8000", server.Addr)
	assert.Same(t, handler, server.Handler)
	assert.Equal(t, readHeaderTimeout, server.ReadHeaderTimeout)
	assert.Equal(t, readTimeout, server.ReadTimeout)
	assert.Equal(t, writeTimeout, server.WriteTimeout)
	assert.Equal(t, idleTimeout, server.IdleTimeout)
}
