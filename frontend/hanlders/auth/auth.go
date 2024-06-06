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
	router.GET("/register", common.Make(h.handleRegisterGet))
	router.POST("/register", common.Make(h.handleRegisterPost))

	router.GET("/login", common.Make(h.handleLoginGet))
	router.POST("/login", common.Make(h.handleLoginPost))
}

func (h *Handler) handleRegisterGet(c *gin.Context) error {
	return common.Render(c.Writer, c.Request, auth.Register())
}

func (h *Handler) handleRegisterPost(c *gin.Context) error {
	time.Sleep(1500 * time.Millisecond)
	firstname := c.PostForm("firstname")
	lastname := c.PostForm("lastname")
	nickname := c.PostForm("nickname")
	email := c.PostForm("email")
	password := c.PostForm("password")

	fmt.Println("username:", email)
	fmt.Println("firstname:", firstname)
	fmt.Println("lastname:", lastname)
	fmt.Println("nickname:", nickname)
	fmt.Println("password:", password)

	c.Header("HX-Redirect", "/login")
	c.Status(200)

	return nil
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

	c.Header("HX-Redirect", "/")
	c.Status(200)

	return nil
}
