package middleware

import (
	"expense-tracker/backend/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddlewareAllowsConfiguredOrigin(t *testing.T) {
	originalOrigins := config.Envs.CORSAllowedOrigins
	originalCredentials := config.Envs.CORSAllowCredentials
	config.Envs.CORSAllowedOrigins = []string{"https://app.example.com"}
	config.Envs.CORSAllowCredentials = true
	t.Cleanup(func() {
		config.Envs.CORSAllowedOrigins = originalOrigins
		config.Envs.CORSAllowCredentials = originalCredentials
	})

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(CORSMiddleware())
	router.GET("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodGet, "/ping", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Equal(t, "https://app.example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
	assert.Equal(t, "Origin", rr.Header().Get("Vary"))
}

func TestCORSMiddlewareRejectsDisallowedPreflightOrigin(t *testing.T) {
	originalOrigins := config.Envs.CORSAllowedOrigins
	config.Envs.CORSAllowedOrigins = []string{"https://app.example.com"}
	t.Cleanup(func() {
		config.Envs.CORSAllowedOrigins = originalOrigins
	})

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(CORSMiddleware())
	router.OPTIONS("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	assert.Empty(t, rr.Header().Get("Access-Control-Allow-Origin"))
}

func TestCORSMiddlewareAllowsConfiguredPreflightOrigin(t *testing.T) {
	originalOrigins := config.Envs.CORSAllowedOrigins
	originalCredentials := config.Envs.CORSAllowCredentials
	config.Envs.CORSAllowedOrigins = []string{"https://app.example.com"}
	config.Envs.CORSAllowCredentials = true
	t.Cleanup(func() {
		config.Envs.CORSAllowedOrigins = originalOrigins
		config.Envs.CORSAllowCredentials = originalCredentials
	})

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(CORSMiddleware())
	router.OPTIONS("/ping", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodOptions, "/ping", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusNoContent, rr.Code)
	assert.Equal(t, "https://app.example.com", rr.Header().Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "true", rr.Header().Get("Access-Control-Allow-Credentials"))
	assert.Contains(t, rr.Header().Get("Access-Control-Allow-Methods"), "OPTIONS")
}
