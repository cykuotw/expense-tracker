package route

import (
	"expense-tracker/backend/config"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestStoreOAuthSessionUsesConfiguredCookiePolicy(t *testing.T) {
	originalDomain := config.Envs.AuthCookieDomain
	originalSecure := config.Envs.AuthCookieSecure
	originalSameSite := config.Envs.AuthCookieSameSite
	originalMaxAge := config.Envs.ThirdPartySessionMaxAge
	config.Envs.AuthCookieDomain = "api.example.com"
	config.Envs.AuthCookieSecure = false
	config.Envs.AuthCookieSameSite = http.SameSiteLaxMode
	config.Envs.ThirdPartySessionMaxAge = 3600
	t.Cleanup(func() {
		config.Envs.AuthCookieDomain = originalDomain
		config.Envs.AuthCookieSecure = originalSecure
		config.Envs.AuthCookieSameSite = originalSameSite
		config.Envs.ThirdPartySessionMaxAge = originalMaxAge
	})

	gin.SetMode(gin.ReleaseMode)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	storeOAuthSession(c, "google", "state-token", "session-json")

	cookies := rr.Result().Cookies()
	if assert.Len(t, cookies, 2) {
		assert.Equal(t, "oauth_google_state", cookies[0].Name)
		assert.Equal(t, "api.example.com", cookies[0].Domain)
		assert.Equal(t, http.SameSiteLaxMode, cookies[0].SameSite)
		assert.True(t, cookies[0].HttpOnly)

		assert.Equal(t, "oauth_google_session", cookies[1].Name)
		assert.Equal(t, "api.example.com", cookies[1].Domain)
		assert.Equal(t, http.SameSiteLaxMode, cookies[1].SameSite)
		assert.True(t, cookies[1].HttpOnly)
	}
}

func TestLoadOAuthSessionRoundTrip(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)
	c.Request.AddCookie(&http.Cookie{Name: oauthStateCookieName("google"), Value: "state-token"})
	c.Request.AddCookie(&http.Cookie{Name: oauthSessionCookieName("google"), Value: encodeOAuthSession(`{"AuthURL":"https://example.com"}`)})

	state, session, err := loadOAuthSession(c, "google")

	assert.NoError(t, err)
	assert.Equal(t, "state-token", state)
	assert.Equal(t, `{"AuthURL":"https://example.com"}`, session)
}

func TestClearOAuthSessionExpiresCookies(t *testing.T) {
	originalSameSite := config.Envs.AuthCookieSameSite
	config.Envs.AuthCookieSameSite = http.SameSiteLaxMode
	t.Cleanup(func() {
		config.Envs.AuthCookieSameSite = originalSameSite
	})

	gin.SetMode(gin.ReleaseMode)
	rr := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(rr)
	c.Request = httptest.NewRequest(http.MethodGet, "/", nil)

	clearOAuthSession(c, "google")

	cookies := rr.Result().Cookies()
	if assert.Len(t, cookies, 2) {
		assert.Equal(t, -1, cookies[0].MaxAge)
		assert.Equal(t, -1, cookies[1].MaxAge)
	}
}
