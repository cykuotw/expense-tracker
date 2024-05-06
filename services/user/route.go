package user

import (
	"github.com/gin-gonic/gin"
)

type Handler struct {
}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/login", h.handleLogin)
	router.POST("/register", h.handleRegister)
}

func (h *Handler) handleLogin(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "login handler",
	})
}

func (h *Handler) handleRegister(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "register handler",
	})
}
