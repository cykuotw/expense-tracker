package types

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

// DB structure
type Item struct {
	ID        uuid.UUID
	ExpenseID uuid.UUID
	Name      string
	Amount    decimal.Decimal
	Unit      string
	UnitPrice decimal.Decimal
}

// Payload
type ItemPayload struct {
	ItemName  string          `json:"itemName"`
	Amount    decimal.Decimal `json:"amount"`
	Unit      string          `json:"unit"`
	UnitPrice decimal.Decimal `json:"unitPrice"`
}

type ItemUpdatePayload struct {
	ID uuid.UUID `json:"itemId"`
	ItemPayload
}

type ItemResponse struct {
	ItemID       uuid.UUID       `json:"itemId"`
	ItemName     string          `json:"itemName"`
	ItemSubTotal decimal.Decimal `json:"itemSubTotal"`
}
