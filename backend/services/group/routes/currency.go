package group

import (
	"errors"
	"net/http"
	"strings"

	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"

	"github.com/gin-gonic/gin"
)

func validateCurrencySettings(store types.GroupStore, settings types.GroupCurrencySettings) error {
	preview := strings.ToUpper(strings.TrimSpace(settings.SettlementPreviewCurrency))
	if preview == "" || preview != settings.SettlementPreviewCurrency || len(settings.Currencies) == 0 {
		return types.ErrInvalidCurrencySettings
	}
	foundPreview := false
	enabled := 0
	seen := make(map[string]struct{}, len(settings.Currencies))
	for _, item := range settings.Currencies {
		if item.Currency != strings.ToUpper(strings.TrimSpace(item.Currency)) {
			return types.ErrInvalidCurrencySettings
		}
		if _, duplicate := seen[item.Currency]; duplicate {
			return types.ErrInvalidCurrencySettings
		}
		seen[item.Currency] = struct{}{}
		supported, err := store.IsSupportedCurrency(item.Currency)
		if err != nil {
			return err
		}
		if !supported || (item.PreviewRate != nil && !item.PreviewRate.IsPositive()) {
			return types.ErrInvalidCurrencySettings
		}
		if item.EnabledForNewExpenses {
			enabled++
		}
		if item.Currency == preview {
			foundPreview = item.EnabledForNewExpenses
		}
	}
	if !foundPreview || enabled == 0 {
		return types.ErrInvalidCurrencySettings
	}
	return nil
}

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

func (h *Handler) handleGetGroupCurrencySettings(c *gin.Context) {
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	store, ok := h.store.(types.GroupCurrencySettingsStore)
	if !ok {
		utils.WriteError(c, http.StatusInternalServerError, types.ErrInvalidCurrencySettings)
		return
	}
	settings, err := store.GetGroupCurrencySettings(c.Param("groupid"), userID)
	if err != nil {
		if errors.Is(err, types.ErrGroupNotExist) {
			utils.WriteError(c, http.StatusNotFound, err)
		} else {
			utils.WriteError(c, http.StatusInternalServerError, err)
		}
		return
	}
	utils.WriteJSON(c, http.StatusOK, settings)
}

func (h *Handler) handleUpdateGroupCurrencySettings(c *gin.Context) {
	var payload types.GroupCurrencySettings
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	if err := validateCurrencySettings(h.store, payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	store, ok := h.store.(types.GroupCurrencySettingsStore)
	if !ok {
		utils.WriteError(c, http.StatusInternalServerError, types.ErrInvalidCurrencySettings)
		return
	}
	if err := store.UpdateGroupCurrencySettings(c.Param("groupid"), userID, payload); err != nil {
		switch {
		case errors.Is(err, types.ErrGroupNotExist):
			utils.WriteError(c, http.StatusNotFound, err)
		case errors.Is(err, types.ErrInvalidCurrencySettings):
			utils.WriteError(c, http.StatusBadRequest, err)
		default:
			utils.WriteError(c, http.StatusInternalServerError, err)
		}
		return
	}
	settings, err := store.GetGroupCurrencySettings(c.Param("groupid"), userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, settings)
}
