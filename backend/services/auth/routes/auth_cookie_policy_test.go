package route

import (
	"expense-tracker/backend/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestAlignOAuthSessionCookieHeadersAddsSameSite(t *testing.T) {
	originalSameSite := config.Envs.AuthCookieSameSite
	config.Envs.AuthCookieSameSite = http.SameSiteLaxMode
	t.Cleanup(func() {
		config.Envs.AuthCookieSameSite = originalSameSite
	})

	gin.SetMode(gin.ReleaseMode)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Writer.Header().Add("Set-Cookie", "_gothic_session=value; Path=/; Domain=api.example.com; HttpOnly; Secure")
	c.Writer.Header().Add("Set-Cookie", "other_cookie=value; Path=/")

	alignOAuthSessionCookieHeaders(c)

	headers := c.Writer.Header().Values("Set-Cookie")
	if assert.Len(t, headers, 2) {
		assert.Contains(t, headers[0], "SameSite=Lax")
		assert.NotContains(t, headers[1], "SameSite")
	}
}

func TestAlignOAuthSessionCookieHeadersDoesNotDuplicateSameSite(t *testing.T) {
	originalSameSite := config.Envs.AuthCookieSameSite
	config.Envs.AuthCookieSameSite = http.SameSiteLaxMode
	t.Cleanup(func() {
		config.Envs.AuthCookieSameSite = originalSameSite
	})

	gin.SetMode(gin.ReleaseMode)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Writer.Header().Add("Set-Cookie", "_gothic_session=value; Path=/; SameSite=Lax")

	alignOAuthSessionCookieHeaders(c)

	headers := c.Writer.Header().Values("Set-Cookie")
	if assert.Len(t, headers, 1) {
		assert.Equal(t, "_gothic_session=value; Path=/; SameSite=Lax", headers[0])
	}
}
