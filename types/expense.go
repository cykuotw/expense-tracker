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
	GetExpenseTypeById(id uuid.UUID) (string, error)
	GetItemsByExpenseID(expenseID string) ([]*Item, error)
	GetLedgersByExpenseID(expenseID string) ([]*Ledger, error)
	GetLedgerUnsettledFromGroup(groupID string) ([]*Ledger, error)
	SettleExpenseByGroupId(groupId string) error

	UpdateExpense(expense Expense) error
	DeleteExpense(expense Expense) error
	UpdateExpenseSettleInGroup(groupID string) error
	UpdateItem(item Item) error
	UpdateLedger(ledger Ledger) error

	CreateBalances(groupId string, balances []*Balance) error
	CreateBalanceLedger(balanceIds []uuid.UUID, ledgerIds []uuid.UUID) error
	OutdateBalanceByGroupId(groupId string) error
	GetBalanceByGroupId(groupId string) ([]Balance, error)
	CheckBalanceExistByID(id string) (bool, error)
	SettleBalanceByBalanceId(balanceId string) error
	CheckGroupBallanceAllSettled(groupId string) (bool, error)
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
	SettleTime     time.Time
	SubTotal       decimal.Decimal
	TaxFeeTip      decimal.Decimal
	Total          decimal.Decimal
	Currency       string
	InvoicePicUrl  string
	SplitRule      string
	IsDeleted      bool
	DeleteTime     time.Time
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
	SplitRule      string          `json:"splitRule"`
	Items          []ItemPayload   `json:"items"`
	Ledgers        []LedgerPayload `json:"ledgers"`
}

type ExpenseUpdatePayload struct {
	Description   string                `json:"description"`
	GroupID       uuid.UUID             `json:"groupId"`
	PayByUserId   string                `json:"payByUserId"`
	ExpenseTypeID uuid.UUID             `json:"expTypeId"`
	ProviderName  string                `json:"providerName"`
	SubTotal      decimal.Decimal       `json:"subTotal"`
	TaxFeeTip     decimal.Decimal       `json:"taxFeeTip"`
	Total         decimal.Decimal       `json:"total"`
	Currency      string                `json:"currency"`
	InvoicePicUrl string                `json:"invoiceUrl"`
	SplitRule     string                `json:"splitRule"`
	Items         []ItemUpdatePayload   `json:"items"`
	Ledgers       []LedgerUpdatePayload `json:"ledgers"`
}

type ExpenseResponseBrief struct {
	ExpenseID      uuid.UUID       `json:"expenseId"`
	Description    string          `json:"description"`
	Total          decimal.Decimal `json:"total"`
	ExpenseTime    time.Time       `json:"expenseTime"`
	CurrentUser    string          `json:"currentUser"`
	Currency       string          `json:"currency"`
	IsSettled      bool            `json:"isSettled"`
	PayerUserIDs   []uuid.UUID     `json:"payerUserIds"`
	PayerUsernames []string        `json:"payerUsernames"`
}

type ExpenseResponse struct {
	ID                uuid.UUID        `json:"expenseId"`
	Description       string           `json:"description"`
	CreatedByUserID   uuid.UUID        `json:"createdByUserID"`
	CreatedByUsername string           `json:"createdByUsername"`
	ExpenseTypeId     uuid.UUID        `json:"expenseTypeId"`
	ExpenseType       string           `json:"expenseType"`
	SubTotal          decimal.Decimal  `json:"subTotal"`
	TaxFeeTip         decimal.Decimal  `json:"taxFeeTip"`
	Total             decimal.Decimal  `json:"total"`
	Currency          string           `json:"currency"`
	ExpenseTime       time.Time        `json:"expenseTime"`
	InvoicePicUrl     string           `json:"invoiceUrl"`
	CurrentUser       string           `json:"currentUser"`
	GroupId           string           `json:"groupId"`
	SplitRule         string           `json:"splitRule"`
	Items             []ItemResponse   `json:"items"`
	Ledgers           []LedgerResponse `json:"ledgers"`
}

type ExpenseTypeResponse struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Name     string `json:"name"`
}
