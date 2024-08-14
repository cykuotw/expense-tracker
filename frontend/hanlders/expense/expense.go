package expense

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
	router.GET("/create_expense", common.Make(h.handleCreateNewExpenseGet))
}

func (h *Handler) handleCreateNewExpenseGet(c *gin.Context) error {
	return common.Render(c.Writer, c.Request, index.NewExpense())
}
