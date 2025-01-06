package expense

import (
	"expense-tracker/services/auth"
	"expense-tracker/types"
	"expense-tracker/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleGetUnsettledBalance(c *gin.Context) {
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

	// get balance
	balanceSimplified, err := h.store.GetBalanceByGroupId(groupID)

	// make response
	groupCurrency, err := h.groupStore.GetGroupCurrency(groupID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	var balances []types.BalanceRsp
	for _, balance := range balanceSimplified {
		senderUsername, err := h.userStore.GetUsernameByID(balance.SenderUserID.String())
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		receiverUsername, err := h.userStore.GetUsernameByID(balance.ReceiverUserID.String())
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		res := types.BalanceRsp{
			SenderUserID:     balance.SenderUserID,
			SenderUesrname:   senderUsername,
			ReceiverUserID:   balance.ReceiverUserID,
			ReceiverUsername: receiverUsername,
			Balance:          balance.Share,
		}

		if res.SenderUserID.String() == userID || res.ReceiverUserID.String() == userID {
			balances = append(balances, res)
		}
	}

	response := types.BalanceResponse{
		Currency:    groupCurrency,
		CurrentUser: userID,
		Balances:    balances,
	}

	utils.WriteJSON(c, http.StatusOK, response)
}
