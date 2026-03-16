package route

import (
	"expense-tracker/backend/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSetAuthCookiesUsesConfiguredDomain(t *testing.T) {
	originalDomain := config.Envs.AuthCookieDomain
	originalSecure := config.Envs.AuthCookieSecure
	originalSameSite := config.Envs.AuthCookieSameSite
	config.Envs.AuthCookieDomain = "api.example.com"
	config.Envs.AuthCookieSecure = false
	config.Envs.AuthCookieSameSite = http.SameSiteLaxMode
	t.Cleanup(func() {
		config.Envs.AuthCookieDomain = originalDomain
		config.Envs.AuthCookieSecure = originalSecure
		config.Envs.AuthCookieSameSite = originalSameSite
	})

	gin.SetMode(gin.ReleaseMode)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest("GET", "/", nil)

	setAuthCookies(c, "access", "refresh")

	cookies := rr.Result().Cookies()
	if assert.Len(t, cookies, 2) {
		assert.Equal(t, "api.example.com", cookies[0].Domain)
		assert.False(t, cookies[0].Secure)
		assert.Equal(t, "/", cookies[0].Path)
		assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
	}
}

func TestSetAuthCookiesRespectsForwardedHTTPS(t *testing.T) {
	originalDomain := config.Envs.AuthCookieDomain
	originalSecure := config.Envs.AuthCookieSecure
	originalSameSite := config.Envs.AuthCookieSameSite
	config.Envs.AuthCookieDomain = ""
	config.Envs.AuthCookieSecure = false
	config.Envs.AuthCookieSameSite = http.SameSiteLaxMode
	t.Cleanup(func() {
		config.Envs.AuthCookieDomain = originalDomain
		config.Envs.AuthCookieSecure = originalSecure
		config.Envs.AuthCookieSameSite = originalSameSite
	})

	gin.SetMode(gin.ReleaseMode)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest("GET", "/", nil)
	c.Request.Header.Set("X-Forwarded-Proto", "https")

	setAuthCookies(c, "access", "refresh")

	cookies := rr.Result().Cookies()
	if assert.Len(t, cookies, 2) {
		assert.True(t, cookies[0].Secure)
		assert.True(t, cookies[1].Secure)
		assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
		assert.Equal(t, http.SameSiteLaxMode, cookies[1].SameSite)
	}
}

func TestClearAuthCookiesKeepsCookieScope(t *testing.T) {
	originalDomain := config.Envs.AuthCookieDomain
	originalSecure := config.Envs.AuthCookieSecure
	originalSameSite := config.Envs.AuthCookieSameSite
	config.Envs.AuthCookieDomain = "api.example.com"
	config.Envs.AuthCookieSecure = true
	config.Envs.AuthCookieSameSite = http.SameSiteLaxMode
	t.Cleanup(func() {
		config.Envs.AuthCookieDomain = originalDomain
		config.Envs.AuthCookieSecure = originalSecure
		config.Envs.AuthCookieSameSite = originalSameSite
	})

	gin.SetMode(gin.ReleaseMode)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest("GET", "/", nil)

	clearAuthCookies(c)

	cookies := rr.Result().Cookies()
	if assert.Len(t, cookies, 2) {
		assert.Equal(t, "api.example.com", cookies[0].Domain)
		assert.Equal(t, "/", cookies[0].Path)
		assert.True(t, cookies[0].Secure)
		assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
		assert.Equal(t, -1, cookies[0].MaxAge)
	}
}
