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
	userIDs := make([]string, 0, len(ledgers)*2)
	for _, ledger := range ledgers {
		userIDs = append(userIDs, ledger.LenderUserID.String(), ledger.BorrowerUesrID.String())
	}
	usernames, err := usernamesByIDs(h.userStore, userIDs)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	ledgerRsp := make([]types.LedgerResponse, 0, len(ledgers))
	for _, led := range ledgers {
		ledger := types.LedgerResponse{
			ID:               led.ID.String(),
			LenderUserId:     led.LenderUserID.String(),
			LenderUsername:   usernames[led.LenderUserID.String()],
			BorrowerUserId:   led.BorrowerUesrID.String(),
			BorrowerUsername: usernames[led.BorrowerUesrID.String()],
			Share:            led.Share,
		}
		ledgerRsp = append(ledgerRsp, ledger)
	}
	expenseTypes, err := h.store.GetExpenseType()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	expenseType := ""
	expenseCategory := ""
	for _, item := range expenseTypes {
		if item.ID != expense.ExpenseTypeID {
			continue
		}
		expenseType = item.Name
		expenseCategory = item.Category
		break
	}
	response := types.ExpenseResponse{
		ID:                expense.ID,
		Description:       expense.Description,
		CreatedByUserID:   expense.CreateByUserID,
		CreatedByUsername: user.Username,
		ExpenseTypeId:     expense.ExpenseTypeID,
		ExpenseType:       expenseType,
		ExpenseCategory:   expenseCategory,
		SubTotal:          expense.SubTotal,
		TaxFeeTip:         expense.TaxFeeTip,
		Total:             expense.Total,
		Currency:          expense.Currency,
		ExpenseTime:       expense.ExpenseTime,
		OccurredOn:        expense.OccurredOn,
		CurrentUser:       userID,
		InvoicePicUrl:     expense.InvoicePicUrl,
		GroupId:           expense.GroupID.String(),
		Items:             itemRsp,
		Ledgers:           ledgerRsp,
		SplitRule:         expense.SplitRule,
	}

	utils.WriteJSON(c, http.StatusOK, response)
}
