package expense

import (
	"net/http"
	"time"

	"expense-tracker/backend/services/middleware/extractors"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"github.com/gin-gonic/gin"
)

func (h *Handler) handleReceiptView(c *gin.Context) {
	c.Header("Cache-Control", "private, no-store")
	c.Header("Referrer-Policy", "no-referrer")
	actor := c.GetString("userID")
	if actor == "" || h.userStore == nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidToken)
		return
	}
	user, userErr := h.userStore.GetUserByID(actor)
	if userErr != nil || user == nil || !user.IsActive {
		utils.WriteError(c, http.StatusForbidden, types.ErrPermissionDenied)
		return
	}
	expense, err := extractors.GetExpenseFromStore(c)
	if err != nil || expense.IsDeleted {
		utils.WriteError(c, http.StatusNotFound, types.ErrExpenseNotExist)
		return
	}
	if h.receiptView == nil {
		utils.WriteError(c, http.StatusServiceUnavailable, errReceiptUnavailable)
		return
	}
	receipt, err := h.store.GetExpenseReceiptByRole(expense.ID, "current")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if receipt == nil || receipt.Status != "finalized" {
		utils.WriteError(c, http.StatusNotFound, types.ErrExpenseNotExist)
		return
	}
	url, err := h.receiptView(c.Request.Context(), receipt.RetainedKey)
	if err != nil {
		utils.WriteError(c, http.StatusServiceUnavailable, errReceiptUnavailable)
		return
	}
	utils.WriteJSON(c, http.StatusOK, gin.H{"url": url, "expiresAt": time.Now().UTC().Add(45 * time.Second)})
}
