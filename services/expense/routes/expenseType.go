package expense

import (
	"expense-tracker/types"
	"expense-tracker/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleGetExpenseType(c *gin.Context) {
	expenseTypes, err := h.store.GetExpenseType()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	var response []types.ExpenseTypeResponse
	for _, expexpenseType := range expenseTypes {
		res := types.ExpenseTypeResponse{
			ID:       expexpenseType.ID.String(),
			Category: expexpenseType.Category,
			Name:     expexpenseType.Name,
		}
		response = append(response, res)
	}

	utils.WriteJSON(c, http.StatusOK, response)
}
