package expense

import (
	"bytes"
	"errors"
	"expense-tracker/backend/services/middleware/extractors"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"
	"time"

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
	key, err := uuid.Parse(c.GetHeader("Idempotency-Key"))
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, types.ErrInvalidIdempotencyKey)
		return
	}
	occurredOn, err := resolveCreateOccurredOn(payload.OccurredOn, time.Now())
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
	if err := h.validateGroupParticipants(groupID, payerID); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
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
		AllocationMode: payload.Allocation.Mode,
		OccurredOn:     occurredOn,
	}
	allocations, err := parseExpenseAllocationPayload(expenseID, payload.Allocation)
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	items := make([]types.Item, 0, len(payload.Items))
	for _, itemPayload := range payload.Items {
		items = append(items, types.Item{
			ID:        uuid.New(),
			ExpenseID: expenseID,
			Name:      itemPayload.ItemName,
			Amount:    itemPayload.Amount,
			Unit:      itemPayload.Unit,
			UnitPrice: itemPayload.UnitPrice,
		})
	}

	resultExpenseID := expenseID

	err = h.store.RunInTransaction(func(store types.ExpenseTransactionStore) error {
		groupCurrency, err := store.LockGroupCurrency(payload.GroupID)
		if err != nil {
			return err
		}
		if payload.Currency != "" && payload.Currency != groupCurrency {
			return types.ErrCurrencyMismatch
		}
		expense.Currency = groupCurrency
		if err := store.CheckGroupParticipants(payload.GroupID, expenseParticipantIDs(creatorID, expense, allocations)); err != nil {
			return err
		}
		amountDigits, err := validateExpenseMoney(store, expense, items)
		if err != nil {
			return err
		}
		ledgers, err := deriveExpenseLedgers(expense, allocations, amountDigits)
		if err != nil {
			return err
		}

		fingerprint, err := expenseCreateFingerprint(expense, items, allocations, payload.OccurredOn)
		if err != nil {
			return err
		}
		existing, claimed, err := store.ClaimExpenseCreateIdempotency(types.ExpenseCreateIdempotency{
			CreatorUserID: creatorID, Key: key, RequestFingerprint: fingerprint, ExpenseID: expenseID,
		})
		if err != nil {
			return err
		}
		if !claimed {
			if !bytes.Equal(existing.RequestFingerprint, fingerprint) {
				return types.ErrIdempotencyKeyConflict
			}
			resultExpenseID = existing.ExpenseID
			return nil
		}
		if err := store.CreateExpense(expense); err != nil {
			return err
		}
		for _, item := range items {
			if err := store.CreateItem(item); err != nil {
				return err
			}
		}
		for _, allocation := range allocations {
			if err := store.CreateExpenseAllocation(allocation); err != nil {
				return err
			}
		}
		for _, ledger := range ledgers {
			if err := store.CreateLedger(ledger); err != nil {
				return err
			}
		}
		if err := store.QueueExpenseCreatedNotifications(expense); err != nil {
			return err
		}

		return h.updateBalanceWithStore(store, payload.GroupID)
	})
	if err != nil {
		if errors.Is(err, types.ErrIdempotencyKeyConflict) {
			utils.WriteError(c, http.StatusConflict, err)
			return
		}
		if errors.Is(err, types.ErrCurrencyMismatch) {
			utils.WriteError(c, http.StatusBadRequest, err)
			return
		}
		if errors.Is(err, types.ErrInvalidMoney) || errors.Is(err, types.ErrInvalidAction) || errors.Is(err, types.ErrUnsupportedCurrency) || errors.Is(err, types.ErrGroupParticipantNotAllowed) {
			utils.WriteError(c, http.StatusBadRequest, err)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, map[string]string{"expenseId": resultExpenseID.String()})
}
