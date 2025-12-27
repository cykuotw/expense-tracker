package expense

import (
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleGetUnsettledBalance(c *gin.Context) {
	groupID := c.Param("groupId")
	userID := c.GetString("userID")

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
			ID:               balance.ID,
			SenderUserID:     balance.SenderUserID,
			SenderUesrname:   senderUsername,
			ReceiverUserID:   balance.ReceiverUserID,
			ReceiverUsername: receiverUsername,
			Balance:          balance.Share,
		}

		balances = append(balances, res)
	}

	response := types.BalanceResponse{
		Currency:    groupCurrency,
		CurrentUser: userID,
		Balances:    balances,
	}

	utils.WriteJSON(c, http.StatusOK, response)
}
