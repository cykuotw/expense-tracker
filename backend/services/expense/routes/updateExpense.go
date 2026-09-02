package expense

import (
	"errors"
	"expense-tracker/backend/services/middleware/extractors"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) handleUpdateExpense(c *gin.Context) {
	expense, err := extractors.GetExpenseFromStore(c)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	payload, err := extractors.GetExpenseUpdatePayload(c)
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	if payload.GroupID != expense.GroupID {
		utils.WriteError(c, http.StatusNotFound, types.ErrExpenseNotExist)
		return
	}
	if payload.OccurredOn != nil {
		occurredOn, err := validateOccurredOn(*payload.OccurredOn)
		if err != nil {
			utils.WriteError(c, http.StatusBadRequest, err)
			return
		}
		expense.OccurredOn = occurredOn
	}

	payerID, err := uuid.Parse(payload.PayByUserId)
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	if err := h.validateGroupParticipants(expense.GroupID, payerID); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	items := make([]types.Item, 0, len(payload.Items))
	for _, itemPayload := range payload.Items {
		itemID := itemPayload.ID
		if itemID == uuid.Nil {
			itemID = uuid.New()
		}
		items = append(items, types.Item{
			ID:        itemID,
			ExpenseID: expense.ID,
			Name:      itemPayload.ItemName,
			Amount:    itemPayload.Amount,
			Unit:      itemPayload.Unit,
			UnitPrice: itemPayload.UnitPrice,
		})
	}

	ledgers := make([]types.Ledger, 0, len(payload.Ledgers))
	newLedgers := make([]bool, 0, len(payload.Ledgers))
	for _, ledgerPayload := range payload.Ledgers {
		lenderID, err := uuid.Parse(ledgerPayload.LenderUserID)
		if err != nil {
			utils.WriteError(c, http.StatusBadRequest, err)
			return
		}
		borrowerID, err := uuid.Parse(ledgerPayload.BorrowerUesrID)
		if err != nil {
			utils.WriteError(c, http.StatusBadRequest, err)
			return
		}
		if err := h.validateGroupParticipants(expense.GroupID, lenderID, borrowerID); err != nil {
			utils.WriteError(c, http.StatusBadRequest, err)
			return
		}

		ledgerID, err := uuid.Parse(ledgerPayload.ID)
		if err != nil {
			ledgerID = uuid.Nil
		}
		newLedger := ledgerID == uuid.Nil
		if newLedger {
			ledgerID = uuid.New()
		}
		newLedgers = append(newLedgers, newLedger)
		ledgers = append(ledgers, types.Ledger{
			ID:             ledgerID,
			ExpenseID:      expense.ID,
			LenderUserID:   lenderID,
			BorrowerUesrID: borrowerID,
			Share:          ledgerPayload.Share,
		})
	}

	updatedExpense := *expense
	updatedExpense.Description = payload.Description
	updatedExpense.PayByUserId = payerID
	updatedExpense.ExpenseTypeID = payload.ExpenseTypeID
	updatedExpense.ProviderName = payload.ProviderName
	updatedExpense.SubTotal = payload.SubTotal
	updatedExpense.TaxFeeTip = payload.TaxFeeTip
	updatedExpense.Total = payload.Total
	updatedExpense.Currency = payload.Currency
	updatedExpense.InvoicePicUrl = payload.InvoicePicUrl
	updatedExpense.SplitRule = payload.SplitRule
	updatedExpense.OccurredOn = expense.OccurredOn

	err = h.store.RunInTransaction(func(store types.ExpenseStore) error {
		for index, item := range items {
			if payload.Items[index].ID == uuid.Nil {
				if err := store.CreateItem(item); err != nil {
					return err
				}
				continue
			}
			if err := store.UpdateItem(item); err != nil {
				return err
			}
		}

		for index, ledger := range ledgers {
			if newLedgers[index] {
				if err := store.CreateLedger(ledger); err != nil {
					return err
				}
				continue
			}
			if err := store.UpdateLedger(ledger); err != nil {
				return err
			}
		}

		if err := store.UpdateExpense(updatedExpense); err != nil {
			return err
		}
		return h.updateBalanceWithStore(store, expense.GroupID.String())
	})
	if err != nil {
		if errors.Is(err, types.ErrItemNotExist) || errors.Is(err, types.ErrLedgerNotExist) {
			utils.WriteError(c, http.StatusNotFound, err)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}
