package expense

import (
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"net/http"

	"expense-tracker/backend/services/middleware/extractors"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func receiptRequestKey(c *gin.Context) (uuid.UUID, error) {
	key, err := uuid.Parse(c.GetHeader("Idempotency-Key"))
	if err != nil || key.Version() != 4 {
		return uuid.Nil, types.ErrInvalidIdempotencyKey
	}
	return key, nil
}

func (h *Handler) handleReceiptMutation(c *gin.Context) {
	expense, err := extractors.GetExpenseFromStore(c)
	if err != nil {
		utils.WriteError(c, http.StatusNotFound, err)
		return
	}
	key, err := receiptRequestKey(c)
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	var payload struct {
		Keep  *bool  `json:"keep"`
		Token string `json:"token"`
	}
	if err := utils.ParseJSON(c, &payload); err != nil || payload.Keep == nil {
		utils.WriteError(c, http.StatusBadRequest, errReceiptChoiceInvalid)
		return
	}
	actorID, err := uuid.Parse(c.GetString("userID"))
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidToken)
		return
	}
	if *payload.Keep {
		h.attachReceipt(c, *expense, actorID, key, payload.Token)
		return
	}
	if payload.Token != "" {
		utils.WriteError(c, http.StatusBadRequest, errReceiptChoiceInvalid)
		return
	}
	h.removeReceipt(c, *expense, actorID, key)
}

func (h *Handler) attachReceipt(c *gin.Context, expense types.Expense, actorID, key uuid.UUID, token string) {
	existing, err := h.store.GetExpenseReceiptByRequest(expense.ID, key)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if existing != nil {
		digest := sha256.Sum256([]byte(token))
		expected, err := hex.DecodeString(existing.TokenSHA256)
		if existing.AccountID != actorID || err != nil || subtle.ConstantTimeCompare(expected, digest[:]) != 1 {
			utils.WriteError(c, http.StatusConflict, types.ErrIdempotencyKeyConflict)
			return
		}
		h.writeReceiptReconciliation(c, existing.ID, expense.ID)
		return
	}
	candidate, err := h.prepareReceipt(c.Request.Context(), &types.ReceiptChoice{Keep: true, Token: token}, actorID.String(), expense.ID, key)
	if err != nil {
		status := http.StatusServiceUnavailable
		if errors.Is(err, errReceiptChoiceInvalid) {
			status = http.StatusBadRequest
		}
		utils.WriteError(c, status, err)
		return
	}
	var cleanupID uuid.UUID
	err = h.store.RunInTransaction(func(store types.ExpenseTransactionStore) error {
		if _, err := store.LockGroupCurrency(expense.GroupID.String()); err != nil {
			return err
		}
		if err := store.CheckGroupParticipants(expense.GroupID.String(), []uuid.UUID{actorID}); err != nil {
			return err
		}
		waiting, err := store.GetExpenseReceiptByRoleForUpdate(expense.ID, "candidate")
		if err != nil {
			return err
		}
		current, err := store.GetExpenseReceiptByRoleForUpdate(expense.ID, "current")
		if err != nil {
			return err
		}
		if waiting != nil {
			if waiting.Status != "failed" {
				return types.ErrIdempotencyKeyConflict
			}
			waiting.Role = "cleanup"
			waiting.Status = "deleting"
			if err := store.UpdateExpenseReceipt(*waiting); err != nil {
				return err
			}
			cleanupID = waiting.ID
		}
		if current != nil {
			candidate.Role = "candidate"
		}
		return store.CreateExpenseReceipt(*candidate)
	})
	if err != nil {
		if errors.Is(err, types.ErrIdempotencyKeyConflict) {
			utils.WriteError(c, http.StatusConflict, err)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if cleanupID != uuid.Nil {
		_, _ = h.reconcileReceipt(c.Request.Context(), cleanupID)
	}
	h.writeReceiptReconciliation(c, candidate.ID, expense.ID)
}

func (h *Handler) removeReceipt(c *gin.Context, expense types.Expense, actorID, key uuid.UUID) {
	existing, err := h.store.GetExpenseReceiptByDeleteRequest(expense.ID, key)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if existing != nil {
		if existing.AccountID != actorID {
			utils.WriteError(c, http.StatusConflict, types.ErrIdempotencyKeyConflict)
			return
		}
		h.writeReceiptCleanup(c, expense.ID)
		return
	}
	err = h.store.RunInTransaction(func(store types.ExpenseTransactionStore) error {
		if _, err := store.LockGroupCurrency(expense.GroupID.String()); err != nil {
			return err
		}
		if err := store.CheckGroupParticipants(expense.GroupID.String(), []uuid.UUID{actorID}); err != nil {
			return err
		}
		recorded := false
		for _, role := range []string{"candidate", "current"} {
			receipt, err := store.GetExpenseReceiptByRoleForUpdate(expense.ID, role)
			if err != nil {
				return err
			}
			if receipt == nil {
				continue
			}
			receipt.Role = "cleanup"
			receipt.Status = "deleting"
			if !recorded {
				receipt.DeleteRequestKey = &key
				recorded = true
			}
			if err := store.UpdateExpenseReceipt(*receipt); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	h.writeReceiptCleanup(c, expense.ID)
}

func (h *Handler) writeReceiptCleanup(c *gin.Context, expenseID uuid.UUID) {
	receipts, err := h.store.ListExpenseReceiptsForCleanup(expenseID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	pending := false
	for _, receipt := range receipts {
		result, err := h.reconcileReceipt(c.Request.Context(), receipt.ID)
		if err != nil || result.Status != "deleted" {
			pending = true
		}
	}
	status := http.StatusOK
	state := "deleted"
	if pending {
		status = http.StatusAccepted
		state = "deleting"
	}
	utils.WriteJSON(c, status, gin.H{"receipt": gin.H{"status": state}})
}

func (h *Handler) writeReceiptReconciliation(c *gin.Context, receiptID, expenseID uuid.UUID) {
	receipt, err := h.reconcileReceipt(c.Request.Context(), receiptID)
	if err != nil {
		utils.WriteError(c, http.StatusServiceUnavailable, err)
		return
	}
	status := http.StatusOK
	if receipt.Status == "pending" || receipt.Status == "deleting" {
		status = http.StatusAccepted
	}
	utils.WriteJSON(c, status, gin.H{"expenseId": expenseID.String(), "receipt": receiptSummary(receipt)})
}

func (h *Handler) handleReceiptRetry(c *gin.Context) {
	expense, err := extractors.GetExpenseFromStore(c)
	if err != nil {
		utils.WriteError(c, http.StatusNotFound, err)
		return
	}
	key, err := receiptRequestKey(c)
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	actorID, err := uuid.Parse(c.GetString("userID"))
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidToken)
		return
	}
	if err := h.store.RunInTransaction(func(store types.ExpenseTransactionStore) error {
		if _, err := store.LockGroupCurrency(expense.GroupID.String()); err != nil {
			return err
		}
		return store.CheckGroupParticipants(expense.GroupID.String(), []uuid.UUID{actorID})
	}); err != nil {
		utils.WriteError(c, http.StatusNotFound, err)
		return
	}
	receipt, err := h.store.GetExpenseReceiptByRequest(expense.ID, key)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if receipt == nil {
		receipt, err = h.store.GetExpenseReceiptByDeleteRequest(expense.ID, key)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
	}
	if receipt == nil || receipt.AccountID != actorID {
		utils.WriteError(c, http.StatusNotFound, types.ErrExpenseNotExist)
		return
	}
	if receipt.DeleteRequestKey != nil && *receipt.DeleteRequestKey == key {
		h.writeReceiptCleanup(c, expense.ID)
		return
	}
	h.writeReceiptReconciliation(c, receipt.ID, expense.ID)
}

// Reconcile is safe after a lost response or page reload. It never creates an
// association, and group authorization is checked by route middleware and here.
func (h *Handler) handleReceiptReconcile(c *gin.Context) {
	expense, err := extractors.GetExpenseFromStore(c)
	if err != nil {
		utils.WriteError(c, http.StatusNotFound, err)
		return
	}
	actorID, err := uuid.Parse(c.GetString("userID"))
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidToken)
		return
	}
	if err := h.store.RunInTransaction(func(store types.ExpenseTransactionStore) error {
		if _, err := store.LockGroupCurrency(expense.GroupID.String()); err != nil {
			return err
		}
		return store.CheckGroupParticipants(expense.GroupID.String(), []uuid.UUID{actorID})
	}); err != nil {
		utils.WriteError(c, http.StatusNotFound, err)
		return
	}
	candidate, err := h.store.GetExpenseReceiptByRole(expense.ID, "candidate")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	current, err := h.store.GetExpenseReceiptByRole(expense.ID, "current")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	var state *types.ExpenseReceipt
	if candidate != nil && candidate.Status == "pending" {
		state, err = h.reconcileReceipt(c.Request.Context(), candidate.ID)
	} else if current != nil && current.Status == "pending" {
		state, err = h.reconcileReceipt(c.Request.Context(), current.ID)
	} else if candidate != nil {
		state = candidate
	} else {
		state = current
	}
	if err != nil {
		utils.WriteError(c, http.StatusServiceUnavailable, err)
		return
	}
	cleanups, err := h.store.ListExpenseReceiptsForCleanup(expense.ID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	pendingCleanup := false
	for _, cleanup := range cleanups {
		result, err := h.reconcileReceipt(c.Request.Context(), cleanup.ID)
		if err != nil || result.Status != "deleted" {
			pendingCleanup = true
		}
	}
	status := http.StatusOK
	if (state != nil && state.Status == "pending") || pendingCleanup {
		status = http.StatusAccepted
	}
	if state == nil && pendingCleanup {
		state = &types.ExpenseReceipt{Status: "deleting"}
	}
	utils.WriteJSON(c, status, gin.H{"expenseId": expense.ID.String(), "receipt": receiptSummary(state)})
}
