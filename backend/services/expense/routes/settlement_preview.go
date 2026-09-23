package expense

import (
	"slices"

	"expense-tracker/backend/types"

	"github.com/shopspring/decimal"
)

func (h *Handler) settlementPreview(
	groupID string,
	userID string,
	balances []types.Balance,
	fallbackCurrency string,
) (*types.SettlementPreview, error) {
	settings := types.GroupCurrencySettings{
		SettlementPreviewCurrency: fallbackCurrency,
		Currencies: []types.GroupCurrencySetting{{
			Currency: fallbackCurrency,
		}},
	}
	one := decimal.NewFromInt(1)
	settings.Currencies[0].PreviewRate = &one
	if settingsStore, ok := h.groupStore.(types.GroupCurrencySettingsStore); ok {
		loaded, err := settingsStore.GetGroupCurrencySettings(groupID, userID)
		if err != nil {
			return nil, err
		}
		settings = loaded
	}

	currencies, err := h.groupStore.ListCurrencies()
	if err != nil {
		return nil, err
	}
	minorUnitDigits := int32(2)
	for _, currency := range currencies {
		if currency.Code == settings.SettlementPreviewCurrency {
			minorUnitDigits = int32(currency.MinorUnitDigits)
			break
		}
	}
	rates := make(map[string]*decimal.Decimal, len(settings.Currencies))
	for _, setting := range settings.Currencies {
		rates[setting.Currency] = setting.PreviewRate
	}
	rates[settings.SettlementPreviewCurrency] = &one

	preview := &types.SettlementPreview{
		Currency:              settings.SettlementPreviewCurrency,
		Complete:              true,
		Contributions:         make([]types.SettlementPreviewContribution, 0),
		MissingRateCurrencies: make([]string, 0),
	}
	net := decimal.Zero
	missing := make(map[string]struct{})
	for _, balance := range balances {
		direction := decimal.Zero
		switch userID {
		case balance.ReceiverUserID.String():
			direction = balance.Share
		case balance.SenderUserID.String():
			direction = balance.Share.Neg()
		default:
			continue
		}
		contribution := types.SettlementPreviewContribution{
			BalanceID:      balance.ID,
			SourceCurrency: balance.Currency,
			SourceAmount:   direction,
		}
		rate := rates[balance.Currency]
		if rate == nil || !rate.IsPositive() {
			missing[balance.Currency] = struct{}{}
			preview.Contributions = append(preview.Contributions, contribution)
			continue
		}
		converted := direction.Mul(*rate).Round(minorUnitDigits)
		copiedRate := *rate
		contribution.Rate = &copiedRate
		contribution.PreviewAmount = &converted
		preview.Contributions = append(preview.Contributions, contribution)
		net = net.Add(converted)
	}
	for currency := range missing {
		preview.MissingRateCurrencies = append(preview.MissingRateCurrencies, currency)
	}
	slices.Sort(preview.MissingRateCurrencies)
	if len(preview.MissingRateCurrencies) > 0 {
		preview.Complete = false
		return preview, nil
	}
	preview.NetAmount = &net
	return preview, nil
}
