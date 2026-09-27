package types

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DB structure
type Item struct {
	ID          uuid.UUID
	ExpenseID   uuid.UUID
	Description string
	Quantity    *decimal.Decimal
	Unit        *string
	UnitPrice   *decimal.Decimal
	LineTotal   decimal.Decimal
	Position    int32
}

// Payload
type ItemPayload struct {
	Description string           `json:"description"`
	Quantity    *decimal.Decimal `json:"quantity"`
	Unit        *string          `json:"unit"`
	UnitPrice   *decimal.Decimal `json:"unitPrice"`
	LineTotal   *decimal.Decimal `json:"lineTotal"`

	// ItemName and Amount preserve the legacy API during the expand-and-contract rollout.
	ItemName string           `json:"itemName,omitempty"`
	Amount   *decimal.Decimal `json:"amount,omitempty"`
}

type ItemUpdatePayload struct {
	ID uuid.UUID `json:"itemId"`
	ItemPayload
}

type ItemResponse struct {
	ItemID      uuid.UUID        `json:"itemId"`
	Description string           `json:"description"`
	Quantity    *decimal.Decimal `json:"quantity"`
	Unit        *string          `json:"unit"`
	UnitPrice   *decimal.Decimal `json:"unitPrice"`
	LineTotal   decimal.Decimal  `json:"lineTotal"`
	Position    int32            `json:"position"`

	// ItemName and ItemSubTotal preserve the legacy response during rollout.
	ItemName     string          `json:"itemName"`
	ItemSubTotal decimal.Decimal `json:"itemSubTotal"`
}
