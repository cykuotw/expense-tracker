package types

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Item struct {
	ID         uuid.UUID
	ExpenseID  uuid.UUID
	ProviderID uuid.UUID
	Name       string
	Amount     decimal.Decimal
	Unit       string
	UnitPrice  decimal.Decimal
}

type Provider struct {
	ID   uuid.UUID
	Name string
}
