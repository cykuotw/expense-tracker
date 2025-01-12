package expense

import (
	"expense-tracker/frontend/handlers/common"

	"github.com/gin-gonic/gin"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/create_expense", common.Make(h.handleCreateNewExpenseGet))
	router.POST("/create_expense", common.Make(h.handleCreateNewExpensePost))

	router.GET("/expense/:expenseId", common.Make(h.handleGetExpenseDetail))
	router.GET("/expense/:expenseId/edit", common.Make(h.handleGetExpenseEdit))
	router.PUT("/expense/:expenseId/delete", common.Make(h.handleGetExpenseDelete))
	router.PUT("/update_expense", common.Make(h.handleUpdateExpense))
	router.POST("settle_expense", common.Make(h.handleSettleExpense))

	router.GET("/expense_types", common.Make(h.handleGetExpenseType))
	router.GET("/expense_types/:select", common.Make(h.handleGetExpenseType))
	router.GET("/split_rules", common.Make(h.handleGetSplitRules))
}
