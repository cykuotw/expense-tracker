package expense

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

	"expense-tracker/backend/services/ocr"
	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

var errReceiptUnavailable = errors.New("receipt storage is unavailable")
var errReceiptChoiceInvalid = errors.New("invalid receipt retention choice")

func (h *Handler) prepareReceipt(ctx context.Context, choice *types.ReceiptChoice, userID string, expenseID, requestKey uuid.UUID) (*types.ExpenseReceipt, error) {
	if choice == nil {
		return nil, nil
	}
	if !choice.Keep {
		if choice.Token != "" {
			return nil, errReceiptChoiceInvalid
		}
		return nil, nil
	}
	if !h.receiptStorageEnabled || h.receiptObjects == nil || len(h.receiptSecret) < 32 {
		return nil, errReceiptUnavailable
	}
	temporary, err := ocr.VerifyReceiptToken(h.receiptSecret, choice.Token, time.Now(), userID)
	if err != nil {
		return nil, errReceiptChoiceInvalid
	}
	receipt, err := receiptFromToken(expenseID, requestKey, temporary)
	digest := sha256.Sum256([]byte(choice.Token))
	receipt.TokenSHA256 = hex.EncodeToString(digest[:])
	if err != nil {
		return nil, errReceiptChoiceInvalid
	}
	verifyCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	if err := h.receiptObjects.verifyTemporary(verifyCtx, receipt); err != nil {
		if errors.Is(err, errReceiptObjectMissing) || errors.Is(err, errReceiptObjectMismatch) {
			return nil, errReceiptChoiceInvalid
		}
		return nil, err
	}
	return &receipt, nil
}

func receiptSummary(receipt *types.ExpenseReceipt) *types.ExpenseReceiptSummary {
	if receipt == nil {
		return nil
	}
	return &types.ExpenseReceiptSummary{
		Status: receipt.Status, ContentType: receipt.ContentType,
		ByteSize: receipt.ByteSize, Width: receipt.Width, Height: receipt.Height,
	}
}

// reconcileReceipt serializes S3 work with receipt mutations. A commit failure
// leaves a tracked row whose deterministic destination can be checked on retry.
func (h *Handler) reconcileReceipt(ctx context.Context, receiptID uuid.UUID) (*types.ExpenseReceipt, error) {
	if h.receiptObjects == nil {
		return nil, errReceiptUnavailable
	}
	var result *types.ExpenseReceipt
	var cleanupID uuid.UUID
	err := h.store.RunInTransaction(func(store types.ExpenseTransactionStore) error {
		receipt, err := store.GetExpenseReceiptForUpdate(receiptID)
		if err != nil {
			return err
		}
		if receipt == nil {
			return errReceiptObjectMissing
		}
		if receipt.Status == "finalized" || receipt.Status == "failed" || receipt.Status == "deleted" {
			result = receipt
			return nil
		}
		workCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		defer cancel()
		receipt.AttemptCount++
		if receipt.Status == "deleting" {
			err := h.receiptObjects.deleteRetained(workCtx, *receipt)
			if err == nil {
				receipt.Status = "deleted"
				now := time.Now().UTC()
				receipt.DeletedAt = &now
				receipt.LastErrorCode = ""
			} else {
				receipt.LastErrorCode = "delete_retryable"
			}
		} else if receipt.Status == "pending" {
			err := h.receiptObjects.finalize(workCtx, *receipt)
			if err == nil {
				receipt.Status = "finalized"
				now := time.Now().UTC()
				receipt.FinalizedAt = &now
				receipt.LastErrorCode = ""
				if receipt.Role == "candidate" {
					previous, err := store.GetExpenseReceiptByRoleForUpdate(receipt.ExpenseID, "current")
					if err != nil {
						return err
					}
					if previous != nil {
						previous.Role = "cleanup"
						previous.Status = "deleting"
						if err := store.UpdateExpenseReceipt(*previous); err != nil {
							return err
						}
						cleanupID = previous.ID
					}
					receipt.Role = "current"
				}
			} else if errors.Is(err, errReceiptObjectMissing) || errors.Is(err, errReceiptObjectMismatch) {
				// A mismatched destination must be removed before terminal failure.
				if errors.Is(err, errReceiptObjectMismatch) {
					if cleanupErr := h.receiptObjects.deleteRetained(workCtx, *receipt); cleanupErr != nil {
						receipt.LastErrorCode = "cleanup_retryable"
					} else {
						receipt.Status = "failed"
						receipt.LastErrorCode = "source_unavailable"
					}
				} else {
					receipt.Status = "failed"
					receipt.LastErrorCode = "source_unavailable"
				}
			} else {
				receipt.LastErrorCode = "finalize_retryable"
			}
		}
		if err := store.UpdateExpenseReceipt(*receipt); err != nil {
			return err
		}
		result = receipt
		return nil
	})
	if err != nil {
		return nil, err
	}
	if cleanupID != uuid.Nil {
		_, _ = h.reconcileReceipt(ctx, cleanupID)
	}
	return result, nil
}
