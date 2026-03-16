package middleware

import (
	"expense-tracker/backend/config"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")
		allowed := isAllowedOrigin(origin, config.Envs.CORSAllowedOrigins)
		if allowed {
			c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			c.Writer.Header().Add("Vary", "Origin")
			c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-CSRF-Token")
			if config.Envs.CORSAllowCredentials {
				c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}

		if c.Request.Method == "OPTIONS" {
			if origin != "" && !allowed {
				c.AbortWithStatus(http.StatusForbidden)
				return
			}
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

func isAllowedOrigin(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}

	for _, allowedOrigin := range allowedOrigins {
		if strings.EqualFold(origin, allowedOrigin) {
			return true
		}
	}

	return false
}
