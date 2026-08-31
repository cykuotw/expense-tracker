package expense

import (
	"errors"
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

	order := types.ExpenseListOrder(c.DefaultQuery("order", string(types.ExpenseListOrderNewest)))
	if order != types.ExpenseListOrderNewest && order != types.ExpenseListOrderOldest {
		utils.WriteError(c, http.StatusBadRequest, errors.New("order must be newest or oldest"))
		return
	}
	status := types.ExpenseListStatus(c.DefaultQuery("status", string(types.ExpenseListStatusAll)))
	if status != types.ExpenseListStatusAll && status != types.ExpenseListStatusUnsettled && status != types.ExpenseListStatusSettled {
		utils.WriteError(c, http.StatusBadRequest, errors.New("status must be all, unsettled, or settled"))
		return
	}

	// get expense list wrt page
	expensePage, err := h.store.GetExpenseList(groupIdStr, page, order, status)
	if errors.Is(err, types.ErrNoRemainingExpenses) {
		utils.WriteJSON(c, http.StatusOK, types.ExpenseResponsePage{
			Expenses: []types.ExpenseResponseBrief{},
			HasMore:  false,
		})
		return
	} else if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	currency, err := h.groupStore.GetGroupCurrency(groupIdStr)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	expenseTypes, err := h.store.GetExpenseType()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	typesByID := make(map[uuid.UUID]*types.ExpenseType, len(expenseTypes))
	for _, expenseType := range expenseTypes {
		typesByID[expenseType.ID] = expenseType
	}
	expenseIDs := make([]string, 0, len(expensePage.Expenses))
	for _, expense := range expensePage.Expenses {
		expenseIDs = append(expenseIDs, expense.ID.String())
	}
	ledgersByExpense, err := ledgersByExpenseIDs(h.store, expenseIDs)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	payerIDs := make([]string, 0)
	seenPayerIDs := make(map[string]struct{})
	for _, ledgers := range ledgersByExpense {
		for _, ledger := range ledgers {
			payerID := ledger.LenderUserID.String()
			if _, seen := seenPayerIDs[payerID]; seen {
				continue
			}
			seenPayerIDs[payerID] = struct{}{}
			payerIDs = append(payerIDs, payerID)
		}
	}
	payerNames, err := usernamesByIDs(h.userStore, payerIDs)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	response := make([]types.ExpenseResponseBrief, 0, len(expensePage.Expenses))
	for _, expense := range expensePage.Expenses {
		payerUserIDs := make([]uuid.UUID, 0)
		payerUsernames := make([]string, 0)

		inserted := make(map[string]struct{})
		for _, ledger := range ledgersByExpense[expense.ID.String()] {
			// 2024.01.12 Single payer model
			// just in case there are multiple payers
			_, ok := inserted[ledger.LenderUserID.String()]
			if !ok {
				payerUserIDs = append(payerUserIDs, ledger.LenderUserID)
				username := payerNames[ledger.LenderUserID.String()]
				payerUsernames = append(payerUsernames, username)
				inserted[ledger.LenderUserID.String()] = struct{}{}
			}
		}

		// get ledger detail
		expenseType := typesByID[expense.ExpenseTypeID]
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
			ExpenseTypeID:  expense.ExpenseTypeID,
		}
		if expenseType != nil {
			res.ExpenseType = expenseType.Name
			res.ExpenseCategory = expenseType.Category
		}
		response = append(response, res)
	}

	utils.WriteJSON(c, http.StatusOK, types.ExpenseResponsePage{
		Expenses: response,
		HasMore:  expensePage.HasMore,
	})
}
