package expense

import (
	"expense-tracker/backend/services/middleware/extractors"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleGetExpenseDetail(c *gin.Context) {
	// get expense id from param
	// check expense id exist and get group id
	expenseID := c.Param("expenseId")
	userID := c.GetString("userID")

	expense, err := extractors.GetExpenseFromStore(c)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	user, err := h.userStore.GetUserByID(expense.CreateByUserID.String())
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	username := user.Username
	items, err := h.store.GetItemsByExpenseID(expenseID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	itemRsp := make([]types.ItemResponse, 0, len(items))
	for _, it := range items {
		item := types.ItemResponse{
			ItemID:       it.ID,
			ItemName:     it.Name,
			ItemSubTotal: it.Amount.Mul(it.UnitPrice),
		}
		itemRsp = append(itemRsp, item)
	}
	ledgers, err := h.store.GetLedgersByExpenseID(expenseID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	ledgerRsp := make([]types.LedgerResponse, 0, len(ledgers))
	for _, led := range ledgers {
		lenderUsername, err := h.userStore.GetUsernameByID(led.LenderUserID.String())
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		borrowerUsername, err := h.userStore.GetUsernameByID(led.BorrowerUesrID.String())
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		ledger := types.LedgerResponse{
			ID:               led.ID.String(),
			LenderUserId:     led.LenderUserID.String(),
			LenderUsername:   lenderUsername,
			BorrowerUserId:   led.BorrowerUesrID.String(),
			BorrowerUsername: borrowerUsername,
			Share:            led.Share,
		}
		ledgerRsp = append(ledgerRsp, ledger)
	}
	expenseType, err := h.store.GetExpenseTypeById(expense.ExpenseTypeID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	response := types.ExpenseResponse{
		ID:                expense.ID,
		Description:       expense.Description,
		CreatedByUserID:   expense.CreateByUserID,
		CreatedByUsername: username,
		ExpenseTypeId:     expense.ExpenseTypeID,
		ExpenseType:       expenseType,
		SubTotal:          expense.SubTotal,
		TaxFeeTip:         expense.TaxFeeTip,
		Total:             expense.Total,
		Currency:          expense.Currency,
		ExpenseTime:       expense.ExpenseTime,
		CurrentUser:       userID,
		InvoicePicUrl:     expense.InvoicePicUrl,
		GroupId:           expense.GroupID.String(),
		Items:             itemRsp,
		Ledgers:           ledgerRsp,
		SplitRule:         expense.SplitRule,
	}

	utils.WriteJSON(c, http.StatusOK, response)
}
