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
	actorID, err := uuid.Parse(c.GetString("userID"))
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

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
	if payload.Currency != "" && payload.Currency != expense.Currency {
		utils.WriteError(c, http.StatusBadRequest, types.ErrCurrencyMismatch)
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
	allocations, err := parseExpenseAllocationPayload(expense.ID, payload.Allocation)
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	updatedExpense := *expense
	updatedExpense.Description = payload.Description
	updatedExpense.PayByUserId = payerID
	updatedExpense.ExpenseTypeID = payload.ExpenseTypeID
	updatedExpense.ProviderName = payload.ProviderName
	updatedExpense.SubTotal = payload.SubTotal
	updatedExpense.TaxFeeTip = payload.TaxFeeTip
	updatedExpense.Total = payload.Total
	updatedExpense.Currency = expense.Currency
	updatedExpense.InvoicePicUrl = payload.InvoicePicUrl
	updatedExpense.AllocationMode = payload.Allocation.Mode
	updatedExpense.OccurredOn = expense.OccurredOn

	err = h.store.RunInTransaction(func(store types.ExpenseTransactionStore) error {
		groupCurrency, err := store.LockGroupCurrency(expense.GroupID.String())
		if err != nil {
			return err
		}
		if groupCurrency != expense.Currency {
			return types.ErrCurrencyMismatch
		}
		if err := store.CheckGroupParticipants(expense.GroupID.String(), expenseParticipantIDs(actorID, updatedExpense, allocations)); err != nil {
			return err
		}
		amountDigits, err := validateExpenseMoney(store, updatedExpense, items)
		if err != nil {
			return err
		}
		ledgers, err := deriveExpenseLedgers(updatedExpense, allocations, amountDigits)
		if err != nil {
			return err
		}
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

		if err := store.UpdateExpense(updatedExpense); err != nil {
			return err
		}
		if err := store.ReconcileExpenseAllocationState(expense.ID, payerID, allocations, ledgers); err != nil {
			return err
		}
		return h.updateBalanceWithStore(store, expense.GroupID.String())
	})
	if err != nil {
		if errors.Is(err, types.ErrItemNotExist) || errors.Is(err, types.ErrLedgerNotExist) {
			utils.WriteError(c, http.StatusNotFound, err)
			return
		}
		if errors.Is(err, types.ErrInvalidMoney) || errors.Is(err, types.ErrInvalidAction) || errors.Is(err, types.ErrUnsupportedCurrency) || errors.Is(err, types.ErrCurrencyMismatch) || errors.Is(err, types.ErrGroupParticipantNotAllowed) {
			utils.WriteError(c, http.StatusBadRequest, err)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}
