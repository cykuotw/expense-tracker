package expense

import (
	"expense-tracker/services/auth"
	"expense-tracker/types"
	"expense-tracker/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleSettleExpense(c *gin.Context) {
	// get group id from param
	groupID := c.Param("groupId")

	// get user id from jwt claim
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	// check user have permission for the group
	exist, err := h.groupStore.CheckGroupUserPairExist(groupID, userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if !exist {
		utils.WriteError(c, http.StatusForbidden, types.ErrPermissionDenied)
		return
	}

	// settle group
	err = h.store.UpdateExpenseSettleInGroup(groupID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}
