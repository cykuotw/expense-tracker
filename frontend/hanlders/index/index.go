package index

import (
	"expense-tracker/frontend/hanlders/common"
	"expense-tracker/frontend/views/index"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/", common.MakeMuitiErr(h.handleIndexGet))
}

func (h *Handler) handleIndexGet(c *gin.Context) []error {

	return []error{common.Render(c.Writer, c.Request, index.Index())}
}
