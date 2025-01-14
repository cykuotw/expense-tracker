package expense

import (
	"expense-tracker/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleSettleBalance(c *gin.Context) {
	groupId := c.Param("groupId")
	balanceId := c.Param("balanceId")

	// settle balance
	err := h.store.SettleBalanceByBalanceId(balanceId)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// check all balances in group are settled
	allSettled, err := h.store.CheckGroupBallanceAllSettled(groupId)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// if all balance are settled, settle all the expense in the group
	if allSettled {
		err = h.store.SettleExpenseByGroupId(groupId)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}
