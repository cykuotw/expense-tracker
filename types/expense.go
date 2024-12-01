package types

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ExpenseStore interface {
	CreateExpense(expense Expense) error
	CreateItem(item Item) error
	CreateLedger(ledger Ledger) error

	CheckExpenseExistByID(id string) (bool, error)

	GetExpenseByID(expenseID string) (*Expense, error)
	GetExpenseList(groupID string, page int64) ([]*Expense, error)
	GetExpenseType() ([]*ExpenseType, error)
	GetItemsByExpenseID(expenseID string) ([]*Item, error)
	GetLedgersByExpenseID(expenseID string) ([]*Ledger, error)
	GetLedgerUnsettledFromGroup(expenseID string) ([]*Ledger, error)

	UpdateExpense(expense Expense) error
	UpdateExpenseSettleInGroup(groupID string) error
	UpdateItem(item Item) error
	UpdateLedger(ledger Ledger) error
}

type ExpenseController interface {
	DebtSimplify(ledgers []*Ledger) []*Balance
}

// DB structure
type Expense struct {
	ID             uuid.UUID
	Description    string
	GroupID        uuid.UUID
	CreateByUserID uuid.UUID
	PayByUserId    uuid.UUID
	ExpenseTypeID  uuid.UUID
	CreateTime     time.Time
	UpdateTime     time.Time
	ExpenseTime    time.Time
	ProviderName   string
	IsSettled      bool
	SubTotal       decimal.Decimal
	TaxFeeTip      decimal.Decimal
	Total          decimal.Decimal
	Currency       string
	InvoicePicUrl  string
}

type ExpenseType struct {
	ID       uuid.UUID
	Name     string
	Category string
}

// Payloads
type ExpensePayload struct {
	Description    string          `json:"description"`
	GroupID        string          `json:"groupId"`
	CreateByUserID string          `json:"createByUserId"`
	PayByUserId    string          `json:"payByUserId"`
	ProviderName   string          `json:"providerName"`
	ExpenseTypeID  string          `json:"expTypeId"`
	SubTotal       decimal.Decimal `json:"subTotal"`
	TaxFeeTip      decimal.Decimal `json:"taxFeeTip"`
	Total          decimal.Decimal `json:"total"`
	Currency       string          `json:"currency"`
	InvoicePicUrl  string          `json:"invoiceUrl"`
	Items          []ItemPayload   `json:"items"`
	Ledgers        []LedgerPayload `json:"ledgers"`
}

type ExpenseUpdatePayload struct {
	Description    string                `json:"description"`
	GroupID        uuid.UUID             `json:"groupId"`
	CreateByUserID uuid.UUID             `json:"createByUserId"`
	ExpenseTypeID  uuid.UUID             `json:"expTypeId"`
	ProviderName   string                `json:"providerName"`
	SubTotal       decimal.Decimal       `json:"subTotal"`
	TaxFeeTip      decimal.Decimal       `json:"taxFeeTip"`
	Total          decimal.Decimal       `json:"total"`
	Currency       string                `json:"currency"`
	InvoicePicUrl  string                `json:"invoiceUrl"`
	Items          []ItemUpdatePayload   `json:"items"`
	Ledgers        []LedgerUpdatePayload `json:"ledgers"`
}

type ExpenseResponseBrief struct {
	ExpenseID      uuid.UUID       `json:"expenseId"`
	Description    string          `json:"description"`
	Total          decimal.Decimal `json:"total"`
	ExpenseTime    time.Time       `json:"expenseTime"`
	CurrentUser    string          `json:"currentUser"`
	Currency       string          `json:"currency"`
	PayerUserIDs   []uuid.UUID     `json:"payerUserIds"`
	PayerUsernames []string        `json:"payerUsernames"`
}

type ExpenseResponse struct {
	ID                uuid.UUID       `json:"expenseId"`
	Description       string          `json:"description"`
	CreatedByUserID   uuid.UUID       `json:"createdByUserID"`
	CreatedByUsername string          `json:"createdByUsername"`
	ExpenseTypeId     uuid.UUID       `json:"expenseTypeId"`
	SubTotal          decimal.Decimal `json:"subTotal"`
	TaxFeeTip         decimal.Decimal `json:"taxFeeTip"`
	Total             decimal.Decimal `json:"total"`
	Currency          string          `json:"currency"`
	ExpenseTime       time.Time       `json:"expenseTime"`
	Items             []ItemResponse  `json:"items"`
}

type ExpenseTypeResponse struct {
	ID       string
	Category string
	Name     string
}
