package route

import (
	"expense-tracker/backend/config"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	authCookiePath     = "/"
	authCookieHTTPOnly = true
)

func setAuthCookies(c *gin.Context, accessToken string, refreshToken string) {
	http.SetCookie(c.Writer, buildAuthCookie("access_token", accessToken, int(config.Envs.JWTExpirationInSeconds), c))
	http.SetCookie(c.Writer, buildAuthCookie("refresh_token", refreshToken, int(config.Envs.RefreshJWTExpirationInSeconds), c))
}

func clearAuthCookies(c *gin.Context) {
	http.SetCookie(c.Writer, buildAuthCookie("access_token", "", -1, c))
	http.SetCookie(c.Writer, buildAuthCookie("refresh_token", "", -1, c))
}

func resolveAuthCookieSecure(c *gin.Context) bool {
	if config.Envs.AuthCookieSecure {
		return true
	}

	if c.Request.TLS != nil {
		return true
	}

	return strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

func buildAuthCookie(name string, value string, maxAge int, c *gin.Context) *http.Cookie {
	return &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     authCookiePath,
		Domain:   config.Envs.AuthCookieDomain,
		MaxAge:   maxAge,
		HttpOnly: authCookieHTTPOnly,
		Secure:   resolveAuthCookieSecure(c),
		SameSite: config.Envs.AuthCookieSameSite,
	}
}
