package route

import (
	"expense-tracker/backend/services/auth"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleLogout(c *gin.Context) error {
	if refreshToken, err := c.Cookie("refresh_token"); err == nil {
		if claims, err := auth.ParseTokenString(refreshToken, "refresh"); err == nil && claims.ID != "" {
			_ = h.refreshStore.RevokeRefreshToken(claims.ID)
		}
	}

	clearAuthCookies(c)

	return nil
}
