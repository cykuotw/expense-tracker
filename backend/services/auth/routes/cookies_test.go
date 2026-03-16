package route

import (
	"expense-tracker/backend/config"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSetAuthCookiesUsesConfiguredDomain(t *testing.T) {
	originalDomain := config.Envs.AuthCookieDomain
	originalSecure := config.Envs.AuthCookieSecure
	config.Envs.AuthCookieDomain = ""
	config.Envs.AuthCookieSecure = false
	t.Cleanup(func() {
		config.Envs.AuthCookieDomain = originalDomain
		config.Envs.AuthCookieSecure = originalSecure
	})

	gin.SetMode(gin.ReleaseMode)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest("GET", "/", nil)

	setAuthCookies(c, "access", "refresh")

	cookies := rr.Result().Cookies()
	if assert.Len(t, cookies, 2) {
		assert.Empty(t, cookies[0].Domain)
		assert.False(t, cookies[0].Secure)
		assert.Equal(t, "/", cookies[0].Path)
	}
}

func TestSetAuthCookiesRespectsForwardedHTTPS(t *testing.T) {
	originalDomain := config.Envs.AuthCookieDomain
	originalSecure := config.Envs.AuthCookieSecure
	config.Envs.AuthCookieDomain = ""
	config.Envs.AuthCookieSecure = false
	t.Cleanup(func() {
		config.Envs.AuthCookieDomain = originalDomain
		config.Envs.AuthCookieSecure = originalSecure
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
	}
}
