package group

import (
	"errors"
	"net/http"
	"strings"

	"expense-tracker/backend/services/exchangerate"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"

	"github.com/gin-gonic/gin"
)

type currencyRecommendationPayload struct {
	SourceCurrencies          []string `json:"sourceCurrencies"`
	SettlementPreviewCurrency string   `json:"settlementPreviewCurrency"`
}

func (h *Handler) handleCurrencyRecommendations(c *gin.Context) {
	var payload currencyRecommendationPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	target := strings.ToUpper(strings.TrimSpace(payload.SettlementPreviewCurrency))
	if target == "" || len(payload.SourceCurrencies) == 0 {
		utils.WriteError(c, http.StatusBadRequest, types.ErrInvalidCurrencySettings)
		return
	}
	sources := make([]string, 0, len(payload.SourceCurrencies))
	seen := make(map[string]struct{}, len(payload.SourceCurrencies))
	for _, raw := range payload.SourceCurrencies {
		code := strings.ToUpper(strings.TrimSpace(raw))
		if code != raw {
			utils.WriteError(c, http.StatusBadRequest, types.ErrInvalidCurrencySettings)
			return
		}
		if _, duplicate := seen[code]; duplicate {
			continue
		}
		supported, err := h.store.IsSupportedCurrency(code)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		if !supported {
			utils.WriteError(c, http.StatusBadRequest, types.ErrUnsupportedCurrency)
			return
		}
		seen[code] = struct{}{}
		sources = append(sources, code)
	}
	if _, ok := seen[target]; !ok {
		utils.WriteError(c, http.StatusBadRequest, types.ErrInvalidCurrencySettings)
		return
	}
	result, err := h.rateClient.Recommendations(c.Request.Context(), sources, target)
	if err != nil {
		if errors.Is(err, exchangerate.ErrUnavailable) {
			utils.WriteError(c, http.StatusBadGateway, err)
		} else {
			utils.WriteError(c, http.StatusInternalServerError, err)
		}
		return
	}
	utils.WriteJSON(c, http.StatusOK, result)
}
