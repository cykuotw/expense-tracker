package route

import (
	"github.com/gin-gonic/gin"
)

func (h *Handler) handleLogout(c *gin.Context) error {
	c.SetCookie("access_token", "", -1, "/", "", false, true)

	return nil
}
