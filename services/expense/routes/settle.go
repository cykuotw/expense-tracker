package expense

import (
	"expense-tracker/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleSettleExpense(c *gin.Context) {
	// get group id from param
	groupID := c.Param("groupId")

	// settle group
	err := h.store.UpdateExpenseSettleInGroup(groupID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}
