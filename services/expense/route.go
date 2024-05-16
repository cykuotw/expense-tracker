package expense

import (
	"expense-tracker/types"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store      types.ExpenseStore
	userStore  types.UserStore
	groupStore types.GroupStore
}

func NewHandler(store types.ExpenseStore, userStore types.UserStore, groupStore types.GroupStore) *Handler {
	return &Handler{
		store:      store,
		userStore:  userStore,
		groupStore: groupStore,
	}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/create_expense", h.handleCreateExpense)
	router.GET("/expense_list", h.handleGetExpenseList)
	router.GET("/expense/:expenseId", h.handleGetExpenseDetail)
	router.PUT("/expense/:expenseId", h.handleUpdateExpense)
	router.PUT("/settle_expense/:groupId", h.handleSettleExpense)
	router.GET("/balance/:groupId", h.handleGetUnsettledBalance)
}

func (h *Handler) handleCreateExpense(c *gin.Context)       {}
func (h *Handler) handleGetExpenseList(c *gin.Context)      {}
func (h *Handler) handleGetExpenseDetail(c *gin.Context)    {}
func (h *Handler) handleUpdateExpense(c *gin.Context)       {}
func (h *Handler) handleSettleExpense(c *gin.Context)       {}
func (h *Handler) handleGetUnsettledBalance(c *gin.Context) {}
