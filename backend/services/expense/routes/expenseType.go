package expense

import (
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleGetExpenseType(c *gin.Context) {
	expenseTypes, err := h.store.GetExpenseType()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusOK, expenseTypeResponses(expenseTypes))
}
