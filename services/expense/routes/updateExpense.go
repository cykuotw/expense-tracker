package expense

import (
	"expense-tracker/services/auth"
	"expense-tracker/types"
	"expense-tracker/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) handleUpdateExpense(c *gin.Context) {
	// get expense id from param
	// check expense id exist and get group id
	expenseID := c.Param("expenseId")

	exist, err := h.store.CheckExpenseExistByID(expenseID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if !exist {
		utils.WriteError(c, http.StatusBadRequest, types.ErrExpenseNotExist)
		return
	}

	expense, err := h.store.GetExpenseByID(expenseID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// get expense update paylaod from body
	var payload types.ExpenseUpdatePayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	// extract userid from jwt, check userid is permitted for the group
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	exist, err = h.groupStore.CheckGroupUserPairExist(payload.GroupID.String(), userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if !exist {
		utils.WriteError(c, http.StatusForbidden, types.ErrPermissionDenied)
		return
	}

	// update items
	items := payload.Items
	for _, it := range items {
		item := types.Item{
			ID:        it.ID,
			ExpenseID: expense.ID,
			Name:      it.ItemName,
			Amount:    it.Amount,
			Unit:      it.Unit,
			UnitPrice: it.UnitPrice,
		}
		if it.ID == uuid.Nil {
			item.ID = uuid.New()

			err := h.store.CreateItem(item)
			if err != nil {
				utils.WriteError(c, http.StatusInternalServerError, err)
				return
			}
		} else {
			err := h.store.UpdateItem(item)
			if err != nil {
				utils.WriteError(c, http.StatusInternalServerError, err)
				return
			}
		}
	}

	// update ledgers
	ledgers := payload.Ledgers
	for _, led := range ledgers {
		lenderID, err := uuid.Parse(led.LenderUserID)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		borrowerID, err := uuid.Parse(led.BorrowerUesrID)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		ledgerId, err := uuid.Parse(led.ID)
		if err != nil {
			ledgerId = uuid.Nil
		}
		ledger := types.Ledger{
			ID:             ledgerId,
			ExpenseID:      expense.ID,
			LenderUserID:   lenderID,
			BorrowerUesrID: borrowerID,
			Share:          led.Share,
		}

		if ledger.ID == uuid.Nil {
			ledger.ID = uuid.New()

			err := h.store.CreateLedger(ledger)
			if err != nil {
				utils.WriteError(c, http.StatusInternalServerError, err)
				return
			}
		} else {
			err := h.store.UpdateLedger(ledger)
			if err != nil {
				utils.WriteError(c, http.StatusInternalServerError, err)
				return
			}
		}

	}

	// update expense
	id := expense.ID
	expense = &types.Expense{
		ID:            id,
		Description:   payload.Description,
		GroupID:       payload.GroupID,
		ExpenseTypeID: payload.ExpenseTypeID,
		ProviderName:  payload.ProviderName,
		SubTotal:      payload.SubTotal,
		TaxFeeTip:     payload.TaxFeeTip,
		Total:         payload.Total,
		Currency:      payload.Currency,
		InvoicePicUrl: payload.InvoicePicUrl,
		SplitRule:     payload.SplitRule,
	}
	err = h.store.UpdateExpense(*expense)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// update balance
	err = h.updateBalance(payload.GroupID.String())
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}
