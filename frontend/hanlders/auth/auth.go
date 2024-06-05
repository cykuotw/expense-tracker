package auth

import (
	"expense-tracker/frontend/hanlders/common"
	"expense-tracker/frontend/views/auth"
	"fmt"
	"time"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	router.GET("/login", common.Make(h.handleLoginGet))
	router.POST("/login", common.Make(h.handleLoginPost))
}

func (h *Handler) handleLoginGet(c *gin.Context) error {
	return common.Render(c.Writer, c.Request, auth.Login())
}

func (h *Handler) handleLoginPost(c *gin.Context) error {
	time.Sleep(1500 * time.Millisecond)
	username := c.PostForm("username")
	password := c.PostForm("password")

	fmt.Println("username:", username)
	fmt.Println("password:", password)

	c.Header("HX-Redirect", "/home")
	c.Status(200)

	return nil
}
