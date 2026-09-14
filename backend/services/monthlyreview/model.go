package monthlyreview

import "time"

const (
	StateNotClosed   = "not_closed"
	StateUnpublished = "unpublished"
	StateEmpty       = "empty"
	StatePublished   = "published"
)

type Review struct {
	GroupID     string            `json:"groupId"`
	GroupName   string            `json:"groupName"`
	Month       string            `json:"month"`
	State       string            `json:"state"`
	PublishedAt time.Time         `json:"publishedAt,omitzero"`
	Currencies  []CurrencySummary `json:"currencies"`
}

type CurrencySummary struct {
	Currency        string        `json:"currency"`
	Total           string        `json:"total"`
	ExpenseCount    int           `json:"expenseCount"`
	Categories      []NamedAmount `json:"categories"`
	Payers          []UserAmount  `json:"payers"`
	MemberNetTotals []UserAmount  `json:"memberNetTotals"`
}

type NamedAmount struct {
	Name   string `json:"name"`
	Amount string `json:"amount"`
}

type UserAmount struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	Amount   string `json:"amount"`
}

type Expense struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	OccurredOn  string `json:"occurredOn"`
	ExpenseType string `json:"expenseType"`
	Category    string `json:"category"`
	PayerID     string `json:"payerId"`
	PayerName   string `json:"payerName"`
	Total       string `json:"total"`
	Settled     bool   `json:"settled"`
}

type ExpensePage struct {
	Expenses   []Expense `json:"expenses"`
	NextCursor string    `json:"nextCursor,omitempty"`
}

type PublicationResult struct {
	Published  int `json:"published"`
	Deliveries int `json:"deliveries"`
}

type Trend struct {
	GroupID           string          `json:"groupId"`
	GroupName         string          `json:"groupName"`
	StartMonth        string          `json:"startMonth"`
	EndMonth          string          `json:"endMonth"`
	LatestReportMonth string          `json:"latestReportMonth,omitempty"`
	Currencies        []TrendCurrency `json:"currencies"`
}

type TrendCurrency struct {
	Currency string       `json:"currency"`
	Months   []TrendMonth `json:"months"`
}

type TrendMonth struct {
	Month        string `json:"month"`
	Total        string `json:"total"`
	ExpenseCount int    `json:"expenseCount"`
}
