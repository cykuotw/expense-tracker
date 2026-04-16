package middleware

import (
	"crypto/rand"
	"encoding/base64"
	"expense-tracker/backend/config"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"
	"net/url"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	CSRFCookieName = "csrf_token"
	CSRFHeaderName = "X-CSRF-Token"
)

func CSRFMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !requiresCSRFProtection(c.Request.Method) {
			c.Next()
			return
		}

		if !isTrustedBrowserOrigin(c.Request) {
			utils.WriteError(c, http.StatusForbidden, types.ErrInvalidCSRFToken)
			c.Abort()
			return
		}

		csrfHeader := c.GetHeader(CSRFHeaderName)
		csrfCookie, err := c.Cookie(CSRFCookieName)
		if err != nil || csrfHeader == "" || csrfHeader != csrfCookie {
			utils.WriteError(c, http.StatusForbidden, types.ErrInvalidCSRFToken)
			c.Abort()
			return
		}

		c.Next()
	}
}

func IssueCSRFToken(c *gin.Context) (string, error) {
	if token, err := c.Cookie(CSRFCookieName); err == nil && token != "" {
		setCSRFCookie(c, token)
		return token, nil
	}

	token, err := generateCSRFToken()
	if err != nil {
		return "", err
	}

	setCSRFCookie(c, token)
	return token, nil
}

func setCSRFCookie(c *gin.Context, token string) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     CSRFCookieName,
		Value:    token,
		Path:     "/",
		Domain:   config.Envs.AuthCookieDomain,
		HttpOnly: false,
		Secure:   shouldUseSecureCookies(c),
		SameSite: config.Envs.AuthCookieSameSite,
	})
}

func shouldUseSecureCookies(c *gin.Context) bool {
	if config.Envs.AuthCookieSecure {
		return true
	}

	if c.Request.TLS != nil {
		return true
	}

	return strings.EqualFold(c.GetHeader("X-Forwarded-Proto"), "https")
}

func generateCSRFToken() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(bytes), nil
}

func requiresCSRFProtection(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func isTrustedBrowserOrigin(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin != "" {
		return strings.EqualFold(origin, config.Envs.FrontendOrigin)
	}

	referer := r.Header.Get("Referer")
	if referer == "" {
		return false
	}

	parsed, err := url.Parse(referer)
	if err != nil {
		return false
	}

	refererOrigin := parsed.Scheme + "://" + parsed.Host
	return strings.EqualFold(refererOrigin, config.Envs.FrontendOrigin)
}
