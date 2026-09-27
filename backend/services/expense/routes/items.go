package expense

import (
	"strings"

	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func normalizeCreateItems(expenseID uuid.UUID, payloads []types.ItemPayload) ([]types.Item, error) {
	items := make([]types.Item, 0, len(payloads))
	for position, payload := range payloads {
		item, err := normalizeItem(expenseID, uuid.New(), int32(position), payload)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func normalizeUpdateItems(expenseID uuid.UUID, payloads []types.ItemUpdatePayload) ([]types.Item, error) {
	items := make([]types.Item, 0, len(payloads))
	seen := make(map[uuid.UUID]struct{}, len(payloads))
	for position, payload := range payloads {
		itemID := payload.ID
		if itemID == uuid.Nil {
			itemID = uuid.New()
		} else if _, exists := seen[itemID]; exists {
			return nil, types.ErrInvalidMoney
		}
		seen[itemID] = struct{}{}
		item, err := normalizeItem(expenseID, itemID, int32(position), payload.ItemPayload)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, nil
}

func normalizeItem(expenseID, itemID uuid.UUID, position int32, payload types.ItemPayload) (types.Item, error) {
	description := strings.TrimSpace(payload.Description)
	legacyDescription := strings.TrimSpace(payload.ItemName)
	if description == "" {
		description = legacyDescription
	} else if legacyDescription != "" && description != legacyDescription {
		return types.Item{}, types.ErrInvalidMoney
	}

	quantity := payload.Quantity
	if quantity == nil {
		quantity = payload.Amount
	} else if payload.Amount != nil && !quantity.Equal(*payload.Amount) {
		return types.Item{}, types.ErrInvalidMoney
	}

	unit := normalizedOptionalString(payload.Unit)
	unitPrice := copyDecimal(payload.UnitPrice)
	lineTotal := copyDecimal(payload.LineTotal)
	usesLegacyContract := strings.TrimSpace(payload.Description) == "" && payload.LineTotal == nil
	if lineTotal == nil && usesLegacyContract && quantity != nil && unitPrice != nil {
		derived := quantity.Mul(*unitPrice).Round(3)
		lineTotal = &derived
	}
	if description == "" || lineTotal == nil {
		return types.Item{}, types.ErrInvalidMoney
	}

	return types.Item{
		ID: itemID, ExpenseID: expenseID, Description: description,
		Quantity: copyDecimal(quantity), Unit: unit, UnitPrice: unitPrice,
		LineTotal: *lineTotal, Position: position,
	}, nil
}

func normalizedOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil
	}
	return &normalized
}

func copyDecimal(value *decimal.Decimal) *decimal.Decimal {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}
