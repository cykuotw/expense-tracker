package expense

import (
	"expense-tracker/services/auth"
	"expense-tracker/types"
	"expense-tracker/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleGetExpenseDetail(c *gin.Context) {
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

	groupID := expense.GroupID.String()

	// extract userid from jwt, check userid is permitted for the group
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	exist, err = h.groupStore.CheckGroupUserPairExist(groupID, userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if !exist {
		utils.WriteError(c, http.StatusForbidden, types.ErrPermissionDenied)
		return
	}

	// get expense
	user, err := h.userStore.GetUserByID(expense.CreateByUserID.String())
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	username := user.Username
	items, err := h.store.GetItemsByExpenseID(expenseID)
	var itemRsp []types.ItemResponse
	for _, it := range items {
		item := types.ItemResponse{
			ItemID:       it.ID,
			ItemName:     it.Name,
			ItemSubTotal: it.Amount.Mul(it.UnitPrice),
		}
		itemRsp = append(itemRsp, item)
	}
	ledgers, err := h.store.GetLedgersByExpenseID(expenseID)
	var ledgerRsp []types.LedgerResponse
	for _, led := range ledgers {
		lenderUsername, _ := h.userStore.GetUsernameByID(led.LenderUserID.String())
		borrowerUsername, _ := h.userStore.GetUsernameByID(led.BorrowerUesrID.String())
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
	expenseType, _ := h.store.GetExpenseTypeById(expense.ExpenseTypeID)
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
