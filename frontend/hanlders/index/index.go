package index

import (
	"fmt"

	"expense-tracker/frontend/hanlders/common"
	"expense-tracker/frontend/views/index"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *gin.Engine) {
	router.GET("/", common.Make(h.handleIndexGet))
}

func (h *Handler) handleIndexGet(c *gin.Context) error {
	fmt.Println("This is index")
	return common.Render(c.Writer, c.Request, index.Index())
}
