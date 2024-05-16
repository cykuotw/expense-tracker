package types

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ExpenseStore interface {
}

type Expense struct {
	ID             uuid.UUID
	Description    string
	GroupID        uuid.UUID
	CreateByUserID uuid.UUID
	ExpenseTypeID  uuid.UUID
	IsSettled      bool
	SubTotal       decimal.Decimal
	TaxFeeTip      decimal.Decimal
	Total          decimal.Decimal
	Currency       string
	InvoicePicUrl  string
}

type ExpenseType struct {
	ID         uuid.UUID
	CategoryID uuid.UUID
	Name       string
}

type Category struct {
	ID   uuid.UUID
	Name string
}

type Ledger struct {
	ID             uuid.UUID
	ExpenseID      uuid.UUID
	LenderUserID   uuid.UUID
	BorrowerUesrID uuid.UUID
	Share          decimal.Decimal
}
