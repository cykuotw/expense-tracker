package expense

import (
	"expense-tracker/backend/services/middleware/extractors"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) handleCreateExpense(c *gin.Context) {
	userID := c.GetString("userID")
	payload, err := extractors.GetExpensePayload(c)
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	// create expense
	expenseID := uuid.New()
	groupID, err := uuid.Parse(payload.GroupID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	creatorID, err := uuid.Parse(userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	payerID, err := uuid.Parse(payload.PayByUserId)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	expTypeID, err := uuid.Parse(payload.ExpenseTypeID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	expense := types.Expense{
		ID:             expenseID,
		Description:    payload.Description,
		GroupID:        groupID,
		CreateByUserID: creatorID,
		PayByUserId:    payerID,
		ExpenseTypeID:  expTypeID,
		ProviderName:   payload.ProviderName,
		IsSettled:      false,
		SubTotal:       payload.SubTotal,
		TaxFeeTip:      payload.TaxFeeTip,
		Total:          payload.Total,
		Currency:       payload.Currency,
		InvoicePicUrl:  payload.InvoicePicUrl,
		SplitRule:      payload.SplitRule,
	}

	err = h.store.CreateExpense(expense)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// create items
	for _, itemPayload := range payload.Items {
		// create provider if not exist
		item := types.Item{
			ID:        uuid.New(),
			ExpenseID: expenseID,
			Name:      itemPayload.ItemName,
			Amount:    itemPayload.Amount,
			Unit:      itemPayload.Unit,
			UnitPrice: itemPayload.UnitPrice,
		}
		err = h.store.CreateItem(item)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
	}

	// create ledgers
	for _, ledgerPayload := range payload.Ledgers {
		lenderUserId, err := uuid.Parse(ledgerPayload.LenderUserID)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		borrowerUserId, err := uuid.Parse(ledgerPayload.BorrowerUesrID)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		id := uuid.New()
		ledger := types.Ledger{
			ID:             id,
			ExpenseID:      expenseID,
			LenderUserID:   lenderUserId,
			BorrowerUesrID: borrowerUserId,
			Share:          ledgerPayload.Share,
		}

		err = h.store.CreateLedger(ledger)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
	}

	// update balance
	err = h.updateBalance(payload.GroupID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, map[string]string{"expenseId": expenseID.String()})
}
