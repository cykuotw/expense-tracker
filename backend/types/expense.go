package types

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type ExpenseListOrder string

const (
	ExpenseListOrderNewest ExpenseListOrder = "newest"
	ExpenseListOrderOldest ExpenseListOrder = "oldest"
)

type ExpenseListStatus string

const (
	ExpenseListStatusAll       ExpenseListStatus = "all"
	ExpenseListStatusUnsettled ExpenseListStatus = "unsettled"
	ExpenseListStatusSettled   ExpenseListStatus = "settled"
)

type ExpenseListPage struct {
	Expenses []*Expense
	HasMore  bool
}

type ExpenseAllocationMode string

const (
	ExpenseAllocationEqual      ExpenseAllocationMode = "equal"
	ExpenseAllocationExact      ExpenseAllocationMode = "exact"
	ExpenseAllocationPercentage ExpenseAllocationMode = "percentage"
	ExpenseAllocationAdjustment ExpenseAllocationMode = "adjustment"
)

type ExpenseAllocation struct {
	ExpenseID             uuid.UUID
	UserID                uuid.UUID
	Amount                *decimal.Decimal
	PercentageBasisPoints *int32
}

type ExpenseAllocationParticipantPayload struct {
	UserID                string           `json:"userId"`
	Amount                *decimal.Decimal `json:"amount,omitempty"`
	PercentageBasisPoints *int32           `json:"percentageBasisPoints,omitempty"`
}

type ExpenseAllocationPayload struct {
	Mode         ExpenseAllocationMode                 `json:"mode"`
	Participants []ExpenseAllocationParticipantPayload `json:"participants"`
}

type ExpenseTransactionStore interface {
	LockGroupCurrency(groupID string) (string, error)
	CheckGroupParticipants(groupID string, userIDs []uuid.UUID) error
	GetCurrencyAmountDigits(currency string) (int32, error)

	CreateExpense(expense Expense) error
	CreateItem(item Item) error
	CreateLedger(ledger Ledger) error
	CreateExpenseAllocation(allocation ExpenseAllocation) error
	ClaimExpenseCreateIdempotency(record ExpenseCreateIdempotency) (existing ExpenseCreateIdempotency, claimed bool, err error)
	QueueExpenseCreatedNotifications(expense Expense) error
	UpdateExpense(expense Expense) error
	UpdateItem(item Item) error
	UpdateLedger(ledger Ledger) error
	ReconcileExpenseAllocationState(expenseID, payerID uuid.UUID, allocations []ExpenseAllocation, ledgers []Ledger) error
	GetLedgerUnsettledFromGroup(groupID string) ([]*Ledger, error)
	CreateBalances(groupId string, balances []*Balance) error
	CreateBalanceLedger(balanceIds []uuid.UUID, ledgerIds []uuid.UUID) error
	OutdateBalanceByGroupId(groupId string) error
}

type ExpenseStore interface {
	RunInTransaction(func(ExpenseTransactionStore) error) error

	CheckExpenseExistByID(id string) (bool, error)

	GetExpenseByID(expenseID string) (*Expense, error)
	GetExpenseList(groupID string, page int64, order ExpenseListOrder, status ExpenseListStatus) (*ExpenseListPage, error)
	GetExpenseType() ([]*ExpenseType, error)
	GetExpenseTypeById(id uuid.UUID) (string, error)
	GetItemsByExpenseID(expenseID string) ([]*Item, error)
	GetLedgersByExpenseID(expenseID string) ([]*Ledger, error)
	GetExpenseAllocationsByExpenseID(expenseID string) ([]ExpenseAllocation, error)
	GetLedgerUnsettledFromGroup(groupID string) ([]*Ledger, error)
	SettleExpenseByGroupId(groupId string) error

	DeleteExpense(expense Expense) error
	UpdateExpenseSettleInGroup(groupID string) error

	GetBalanceByGroupId(groupId string) ([]Balance, error)
	CreateBalances(groupId string, balances []*Balance) error
	CreateBalanceLedger(balanceIds []uuid.UUID, ledgerIds []uuid.UUID) error
	OutdateBalanceByGroupId(groupId string) error
	CheckBalanceExistByID(id string) (bool, error)
	SettleBalanceByBalanceId(groupID string, balanceID string) error
	CheckGroupBallanceAllSettled(groupId string) (bool, error)
}

type ExpenseCreateIdempotency struct {
	CreatorUserID      uuid.UUID
	Key                uuid.UUID
	RequestFingerprint []byte
	ExpenseID          uuid.UUID
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
	OccurredOn     string
	ProviderName   string
	IsSettled      bool
	SettleTime     time.Time
	SubTotal       decimal.Decimal
	TaxFeeTip      decimal.Decimal
	Total          decimal.Decimal
	Currency       string
	InvoicePicUrl  string
	AllocationMode ExpenseAllocationMode
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
	Description    string                   `json:"description"`
	GroupID        string                   `json:"groupId"`
	CreateByUserID string                   `json:"createByUserId"`
	PayByUserId    string                   `json:"payByUserId"`
	ProviderName   string                   `json:"providerName"`
	ExpenseTypeID  string                   `json:"expTypeId"`
	SubTotal       decimal.Decimal          `json:"subTotal"`
	TaxFeeTip      decimal.Decimal          `json:"taxFeeTip"`
	Total          decimal.Decimal          `json:"total"`
	Currency       string                   `json:"currency"`
	InvoicePicUrl  string                   `json:"invoiceUrl"`
	OccurredOn     *string                  `json:"occurredOn"`
	Items          []ItemPayload            `json:"items"`
	Allocation     ExpenseAllocationPayload `json:"allocation"`
}

type ExpenseUpdatePayload struct {
	Description   string                   `json:"description"`
	GroupID       uuid.UUID                `json:"groupId"`
	PayByUserId   string                   `json:"payByUserId"`
	ExpenseTypeID uuid.UUID                `json:"expTypeId"`
	ProviderName  string                   `json:"providerName"`
	SubTotal      decimal.Decimal          `json:"subTotal"`
	TaxFeeTip     decimal.Decimal          `json:"taxFeeTip"`
	Total         decimal.Decimal          `json:"total"`
	Currency      string                   `json:"currency"`
	InvoicePicUrl string                   `json:"invoiceUrl"`
	OccurredOn    *string                  `json:"occurredOn"`
	Items         []ItemUpdatePayload      `json:"items"`
	Allocation    ExpenseAllocationPayload `json:"allocation"`
}

type ExpenseResponseBrief struct {
	ExpenseID       uuid.UUID       `json:"expenseId"`
	Description     string          `json:"description"`
	Total           decimal.Decimal `json:"total"`
	ExpenseTime     time.Time       `json:"expenseTime"`
	OccurredOn      string          `json:"occurredOn"`
	CurrentUser     string          `json:"currentUser"`
	Currency        string          `json:"currency"`
	IsSettled       bool            `json:"isSettled"`
	PayerUserIDs    []uuid.UUID     `json:"payerUserIds"`
	PayerUsernames  []string        `json:"payerUsernames"`
	ExpenseTypeID   uuid.UUID       `json:"expenseTypeId"`
	ExpenseType     string          `json:"expenseType"`
	ExpenseCategory string          `json:"expenseCategory"`
}

type ExpenseResponsePage struct {
	Expenses []ExpenseResponseBrief `json:"expenses"`
	HasMore  bool                   `json:"hasMore"`
}

// GroupOverviewResponse is the bounded initial read for the group page.
// Additional expense pages remain independently pageable.
type GroupOverviewResponse struct {
	Group    GetGroupResponse    `json:"group"`
	Balance  BalanceResponse     `json:"balance"`
	Expenses ExpenseResponsePage `json:"expenses"`
}

type ExpenseResponse struct {
	ID                uuid.UUID                `json:"expenseId"`
	Description       string                   `json:"description"`
	CreatedByUserID   uuid.UUID                `json:"createdByUserID"`
	CreatedByUsername string                   `json:"createdByUsername"`
	ExpenseTypeId     uuid.UUID                `json:"expenseTypeId"`
	ExpenseType       string                   `json:"expenseType"`
	ExpenseCategory   string                   `json:"expenseCategory"`
	SubTotal          decimal.Decimal          `json:"subTotal"`
	TaxFeeTip         decimal.Decimal          `json:"taxFeeTip"`
	Total             decimal.Decimal          `json:"total"`
	Currency          string                   `json:"currency"`
	ExpenseTime       time.Time                `json:"expenseTime"`
	OccurredOn        string                   `json:"occurredOn"`
	InvoicePicUrl     string                   `json:"invoiceUrl"`
	CurrentUser       string                   `json:"currentUser"`
	GroupId           string                   `json:"groupId"`
	Allocation        ExpenseAllocationPayload `json:"allocation"`
	Items             []ItemResponse           `json:"items"`
	Ledgers           []LedgerResponse         `json:"ledgers"`
}

type ExpenseTypeResponse struct {
	ID       string `json:"id"`
	Category string `json:"category"`
	Name     string `json:"name"`
}

// CreateExpenseOptionsResponse is the bounded initial read model for the
// Create Expense page. Group is nil when the page is opened without a group.
type CreateExpenseOptionsResponse struct {
	Groups       []GetGroupListResponse `json:"groups"`
	ExpenseTypes []ExpenseTypeResponse  `json:"expenseTypes"`
	Currencies   []Currency             `json:"currencies"`
	Group        *GetGroupResponse      `json:"group"`
}

// EditExpenseOptionsResponse is the bounded initial read model for the Edit
// Expense page.
type EditExpenseOptionsResponse struct {
	Expense      ExpenseResponse        `json:"expense"`
	Groups       []GetGroupListResponse `json:"groups"`
	ExpenseTypes []ExpenseTypeResponse  `json:"expenseTypes"`
	Currencies   []Currency             `json:"currencies"`
	Group        GetGroupResponse       `json:"group"`
}
