package index

import (
	"expense-tracker/frontend/hanlders/common"
	"expense-tracker/frontend/views/index"
	"net/http"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	router.GET("/", common.MakeMuitiErr(h.handleIndexGet))
}

func (h *Handler) handleIndexGet(c *gin.Context) []error {
	_, err := c.Cookie("access_token")
	if err != nil {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
		return []error{err}
	}

	return []error{common.Render(c.Writer, c.Request, index.Index())}
}
