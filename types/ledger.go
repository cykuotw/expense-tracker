package types

import (
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Ledger struct {
	ID             uuid.UUID
	ExpenseID      uuid.UUID
	LenderUserID   uuid.UUID
	BorrowerUesrID uuid.UUID
	Share          decimal.Decimal
}

// payload
type LedgerPayload struct {
	LenderUserID   string          `json:"lenderUserId"`
	BorrowerUesrID string          `json:"borrowerUserId"`
	Share          decimal.Decimal `json:"share"`
}

type LedgerUpdatePayload struct {
	ID uuid.UUID `json:"ledgerId"`
	LedgerPayload
}

type LedgerResponse struct {
	LenderUserId     string          `json:"lenderUserId"`
	LenderUsername   string          `json:"lenderUsername"`
	BorrowerUserId   string          `json:"borrowerUserId"`
	BorrowerUsername string          `json:"borrowerUsername"`
	Share            decimal.Decimal `json:"share"`
}
