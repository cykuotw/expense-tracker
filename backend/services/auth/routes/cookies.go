package route

import (
	"expense-tracker/backend/config"

	"github.com/gin-gonic/gin"
)

const (
	authCookiePath   = "/"
	authCookieDomain = "localhost"
	authCookieSecure = false
	authCookieHTTPOnly = true
)

func setAuthCookies(c *gin.Context, accessToken string, refreshToken string) {
	c.SetCookie(
		"access_token", accessToken,
		int(config.Envs.JWTExpirationInSeconds),
		authCookiePath, authCookieDomain, authCookieSecure, authCookieHTTPOnly,
	)
	c.SetCookie(
		"refresh_token", refreshToken,
		int(config.Envs.RefreshJWTExpirationInSeconds),
		authCookiePath, authCookieDomain, authCookieSecure, authCookieHTTPOnly,
	)
}

func clearAuthCookies(c *gin.Context) {
	c.SetCookie(
		"access_token", "",
		-1,
		authCookiePath, authCookieDomain, authCookieSecure, authCookieHTTPOnly,
	)
	c.SetCookie(
		"refresh_token", "",
		-1,
		authCookiePath, authCookieDomain, authCookieSecure, authCookieHTTPOnly,
	)
}
