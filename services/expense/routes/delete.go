package expense

import (
	"expense-tracker/services/middleware/extractors"
	"expense-tracker/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleDeleteExpense(c *gin.Context) {
	// get expense id from param
	// check expense id exist and get group id
	expense, err := extractors.GetExpenseFromStore(c)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// delete expense
	err = h.store.DeleteExpense(*expense)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// update balance
	err = h.updateBalance(expense.GroupID.String())
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusOK, nil)
}
