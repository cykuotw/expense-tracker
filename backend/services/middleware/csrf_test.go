package middleware

import (
	"encoding/json"
	"expense-tracker/backend/config"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCSRFMiddlewareAllowsValidToken(t *testing.T) {
	originalFrontendOrigin := config.Envs.FrontendOrigin
	config.Envs.FrontendOrigin = "https://app.example.com"
	t.Cleanup(func() {
		config.Envs.FrontendOrigin = originalFrontendOrigin
	})

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(CSRFMiddleware())
	router.POST("/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.Header.Set("Origin", "https://app.example.com")
	req.Header.Set(CSRFHeaderName, "token")
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "token"})
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCSRFMiddlewareRejectsMissingToken(t *testing.T) {
	originalFrontendOrigin := config.Envs.FrontendOrigin
	config.Envs.FrontendOrigin = "https://app.example.com"
	t.Cleanup(func() {
		config.Envs.FrontendOrigin = originalFrontendOrigin
	})

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(CSRFMiddleware())
	router.POST("/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
	var response struct {
		Error string `json:"error"`
		Code  string `json:"code"`
	}
	err := json.Unmarshal(rr.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, types.ErrInvalidCSRFToken.Error(), response.Error)
	assert.Equal(t, "invalid_csrf_token", response.Code)
}

func TestCSRFMiddlewareRejectsUntrustedOrigin(t *testing.T) {
	originalFrontendOrigin := config.Envs.FrontendOrigin
	config.Envs.FrontendOrigin = "https://app.example.com"
	t.Cleanup(func() {
		config.Envs.FrontendOrigin = originalFrontendOrigin
	})

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(CSRFMiddleware())
	router.POST("/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.Header.Set("Origin", "https://evil.example.com")
	req.Header.Set(CSRFHeaderName, "token")
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "token"})
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusForbidden, rr.Code)
}

func TestCSRFMiddlewareAllowsOAuthFormPostWithTrustedOrigin(t *testing.T) {
	originalFrontendOrigin := config.Envs.FrontendOrigin
	config.Envs.FrontendOrigin = "https://app.example.com"
	t.Cleanup(func() {
		config.Envs.FrontendOrigin = originalFrontendOrigin
	})

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(CSRFMiddleware())
	router.POST("/auth/google", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/auth/google", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestIssueCSRFTokenSetsCookieAndReturnsToken(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest(http.MethodGet, "/auth/csrf", nil)

	token, err := IssueCSRFToken(c)
	assert.NoError(t, err)
	assert.NotEmpty(t, token)

	cookies := rr.Result().Cookies()
	if assert.Len(t, cookies, 1) {
		assert.Equal(t, CSRFCookieName, cookies[0].Name)
		assert.Equal(t, token, cookies[0].Value)
	}
}

func TestCSRFMiddlewareRefererFallback(t *testing.T) {
	originalFrontendOrigin := config.Envs.FrontendOrigin
	config.Envs.FrontendOrigin = "https://app.example.com"
	t.Cleanup(func() {
		config.Envs.FrontendOrigin = originalFrontendOrigin
	})

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(CSRFMiddleware())
	router.POST("/login", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest(http.MethodPost, "/login", nil)
	req.Header.Set("Referer", "https://app.example.com/login")
	req.Header.Set(CSRFHeaderName, "token")
	req.AddCookie(&http.Cookie{Name: CSRFCookieName, Value: "token"})
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}

func TestCSRFMiddlewareUsesAPIPathAwareOAuthBypass(t *testing.T) {
	originalAPIPath := config.Envs.APIPath
	originalFrontendOrigin := config.Envs.FrontendOrigin
	config.Envs.APIPath = "/api"
	config.Envs.FrontendOrigin = "https://app.example.com"
	t.Cleanup(func() {
		config.Envs.APIPath = originalAPIPath
		config.Envs.FrontendOrigin = originalFrontendOrigin
	})

	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	router.Use(CSRFMiddleware())
	router.POST("/api/auth/google", func(c *gin.Context) {
		utils.WriteJSON(c, http.StatusOK, nil)
	})

	req := httptest.NewRequest(http.MethodPost, "/api/auth/google", nil)
	req.Header.Set("Origin", "https://app.example.com")
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
}
