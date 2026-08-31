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
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// make response
	groupCurrency, err := h.groupStore.GetGroupCurrency(groupID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	userIDs := make([]string, 0, len(balanceSimplified)*2)
	for _, balance := range balanceSimplified {
		userIDs = append(userIDs, balance.SenderUserID.String(), balance.ReceiverUserID.String())
	}
	usernames, err := usernamesByIDs(h.userStore, userIDs)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	balances := make([]types.BalanceRsp, 0, len(balanceSimplified))
	for _, balance := range balanceSimplified {
		res := types.BalanceRsp{
			ID:               balance.ID,
			SenderUserID:     balance.SenderUserID,
			SenderUesrname:   usernames[balance.SenderUserID.String()],
			ReceiverUserID:   balance.ReceiverUserID,
			ReceiverUsername: usernames[balance.ReceiverUserID.String()],
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
