package expense

import (
	"expense-tracker/backend/services/middleware/extractors"
	"expense-tracker/backend/services/middleware/validation"
	"expense-tracker/backend/types"

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
	router.Use(extractors.ExtractUserIdFromJWT())

	router.POST("/create_expense",
		extractors.ExtractExpensePayload(),
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleCreateExpense)
	router.GET("/expense_list/:groupId",
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleGetExpenseList)
	router.GET("/expense_list/:groupId/:page",
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleGetExpenseList)
	router.GET("/expense_types", h.handleGetExpenseType)
	router.GET("/expense/:expenseId",
		validation.ValidateExpenseExist(h.store),
		extractors.ExtractExpenseFromStore(h.store),
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleGetExpenseDetail)
	router.PUT("/expense/:expenseId",
		validation.ValidateExpenseExist(h.store),
		extractors.ExtractExpenseUpdatePayload(),
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleUpdateExpense)
	router.PUT("/delete_expense/:expenseId",
		validation.ValidateExpenseExist(h.store),
		extractors.ExtractExpenseFromStore(h.store),
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleDeleteExpense)
	router.PUT("/settle_expense/:groupId",
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleSettleExpense)
	router.GET("/balance/:groupId",
		validation.ValidateGroupUserPairExist(h.groupStore),
		h.handleGetUnsettledBalance)
	router.POST("/settle_balance/:groupId/:balanceId",
		validation.ValidateGroupUserPairExist(h.groupStore),
		validation.ValidateBalanceExist(h.store),
		h.handleSettleBalance)
}
