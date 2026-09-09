package group

import (
	"errors"
	"net/http"

	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"

	"github.com/gin-gonic/gin"
)

func (h *Handler) handleListCurrencies(c *gin.Context) {
	currencies, err := h.store.ListCurrencies()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, currencies)
}

func (h *Handler) handleUpdateGroupCurrency(c *gin.Context) {
	var payload types.UpdateGroupCurrencyPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if err := h.store.UpdateGroupCurrency(c.Param("groupid"), userID, payload.Currency); err != nil {
		switch {
		case errors.Is(err, types.ErrGroupNotExist):
			utils.WriteError(c, http.StatusNotFound, err)
		case errors.Is(err, types.ErrGroupCurrencyLocked):
			utils.WriteError(c, http.StatusConflict, err)
		case errors.Is(err, types.ErrUnsupportedCurrency):
			utils.WriteError(c, http.StatusBadRequest, err)
		default:
			utils.WriteError(c, http.StatusInternalServerError, err)
		}
		return
	}

	utils.WriteJSON(c, http.StatusOK, gin.H{"currency": payload.Currency})
}
