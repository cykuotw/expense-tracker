package route

import (
	"expense-tracker/backend/config"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	authCookiePath     = "/"
	authCookieHTTPOnly = true
)

func setAuthCookies(c *gin.Context, accessToken string, refreshToken string) {
	c.SetCookie(
		"access_token", accessToken,
		int(config.Envs.JWTExpirationInSeconds),
		authCookiePath, config.Envs.AuthCookieDomain, resolveAuthCookieSecure(c), authCookieHTTPOnly,
	)
	c.SetCookie(
		"refresh_token", refreshToken,
		int(config.Envs.RefreshJWTExpirationInSeconds),
		authCookiePath, config.Envs.AuthCookieDomain, resolveAuthCookieSecure(c), authCookieHTTPOnly,
	)
}

func clearAuthCookies(c *gin.Context) {
	c.SetCookie(
		"access_token", "",
		-1,
		authCookiePath, config.Envs.AuthCookieDomain, resolveAuthCookieSecure(c), authCookieHTTPOnly,
	)
	c.SetCookie(
		"refresh_token", "",
		-1,
		authCookiePath, config.Envs.AuthCookieDomain, resolveAuthCookieSecure(c), authCookieHTTPOnly,
	)
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
