package auth

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, err := c.Cookie("access_token")
		if err != nil {
			slog.Error("HTTP handler error", "err", err, "path", c.Request.URL.Path)
			c.Redirect(http.StatusTemporaryRedirect, "/login")
		}
		c.Next()
	}
}
