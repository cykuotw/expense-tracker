package expense

import (
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) handleGetExpenseList(c *gin.Context) {
	// get group id, page from param
	groupIdStr := c.Param("groupId")
	if groupIdStr == "" {
		utils.WriteError(c, http.StatusBadRequest, types.ErrGroupNotExist)
		return
	}

	pageStr := c.Param("page")
	var err error
	page := int64(0)
	if pageStr != "" {
		page, err = strconv.ParseInt(pageStr, 10, 0)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
	}

	userID := c.GetString("userID")

	// get expense list wrt page
	expenseList, err := h.store.GetExpenseList(groupIdStr, page)
	if err == types.ErrNoRemainingExpenses {
		utils.WriteJSON(c, http.StatusOK, []types.ExpenseResponseBrief{})
		return
	} else if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	currency, nil := h.groupStore.GetGroupCurrency(groupIdStr)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	var response []types.ExpenseResponseBrief
	for _, expense := range expenseList {
		var payerUserIDs []uuid.UUID
		var payerUsernames []string

		ledgers, err := h.store.GetLedgersByExpenseID(expense.ID.String())
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}

		inserted := make(map[string]interface{})
		for _, ledger := range ledgers {
			// 2024.01.12 Single payer model
			// just in case there are multiple payers
			_, ok := inserted[ledger.LenderUserID.String()]
			if !ok {
				payerUserIDs = append(payerUserIDs, ledger.LenderUserID)
				username, err := h.userStore.GetUsernameByID(ledger.LenderUserID.String())
				if err != nil {
					utils.WriteError(c, http.StatusInternalServerError, err)
					return
				}
				payerUsernames = append(payerUsernames, username)
				inserted[ledger.LenderUserID.String()] = nil
			}
		}

		// get ledger detail
		res := types.ExpenseResponseBrief{
			ExpenseID:      expense.ID,
			Description:    expense.Description,
			Total:          expense.Total,
			ExpenseTime:    expense.ExpenseTime,
			CurrentUser:    userID,
			Currency:       currency,
			IsSettled:      expense.IsSettled,
			PayerUserIDs:   payerUserIDs,
			PayerUsernames: payerUsernames,
		}
		response = append(response, res)
	}

	utils.WriteJSON(c, http.StatusOK, response)
}
