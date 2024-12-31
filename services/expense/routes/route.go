package expense

import (
	"expense-tracker/types"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store      types.ExpenseStore
	userStore  types.UserStore
	groupStore types.GroupStore

	controller types.ExpenseController
}

func NewHandler(store types.ExpenseStore, userStore types.UserStore, groupStore types.GroupStore, controller types.ExpenseController) *Handler {
	return &Handler{
		store:      store,
		userStore:  userStore,
		groupStore: groupStore,

		controller: controller,
	}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/create_expense", h.handleCreateExpense)
	router.GET("/expense_list/:groupId", h.handleGetExpenseList)
	router.GET("/expense_list/:groupId/:page", h.handleGetExpenseList)
	router.GET("/expense_types", h.handleGetExpenseType)
	router.GET("/expense/:expenseId", h.handleGetExpenseDetail)
	router.PUT("/expense/:expenseId", h.handleUpdateExpense)
	router.PUT("/delete_expense/:expenseId", h.handleDeleteExpense)
	router.PUT("/settle_expense/:groupId", h.handleSettleExpense)
	router.GET("/balance/:groupId", h.handleGetUnsettledBalance)
}
