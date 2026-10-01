package expense

import (
	"crypto/sha256"
	"encoding/json"
	"slices"
	"strings"

	"expense-tracker/backend/types"

	"github.com/shopspring/decimal"
)

type canonicalExpenseCreate struct {
	Description    string                       `json:"description"`
	GroupID        string                       `json:"groupId"`
	PayerID        string                       `json:"payByUserId"`
	ExpenseTypeID  string                       `json:"expenseTypeId"`
	ProviderName   string                       `json:"providerName"`
	SubTotal       string                       `json:"subTotal"`
	TaxFeeTip      string                       `json:"taxFeeTip"`
	Total          string                       `json:"total"`
	Currency       string                       `json:"currency"`
	InvoiceURL     string                       `json:"invoiceUrl"`
	AllocationMode types.ExpenseAllocationMode  `json:"allocationMode"`
	OccurredOn     string                       `json:"occurredOn,omitempty"`
	Items          []canonicalExpenseItem       `json:"items"`
	Allocations    []canonicalExpenseAllocation `json:"allocations"`
	ReceiptKeep    bool                         `json:"receiptKeep,omitempty"`
	ReceiptToken   string                       `json:"receiptToken,omitempty"`
}

type canonicalExpenseItem struct {
	Description string  `json:"description"`
	Quantity    *string `json:"quantity"`
	Unit        *string `json:"unit"`
	UnitPrice   *string `json:"unitPrice"`
	LineTotal   string  `json:"lineTotal"`
	Position    int32   `json:"position"`
}

type canonicalExpenseAllocation struct {
	UserID                string `json:"userId"`
	Amount                string `json:"amount,omitempty"`
	PercentageBasisPoints *int32 `json:"percentageBasisPoints,omitempty"`
}

func expenseCreateFingerprint(
	expense types.Expense,
	items []types.Item,
	allocations []types.ExpenseAllocation,
	requestedOccurredOn *string,
	receiptChoice ...*types.ReceiptChoice,
) ([]byte, error) {
	occurredOn := ""
	if requestedOccurredOn != nil {
		occurredOn = expense.OccurredOn
	}
	canonical := canonicalExpenseCreate{
		Description: expense.Description, GroupID: expense.GroupID.String(), PayerID: expense.PayByUserId.String(),
		ExpenseTypeID: expense.ExpenseTypeID.String(), ProviderName: expense.ProviderName,
		SubTotal: expense.SubTotal.String(), TaxFeeTip: expense.TaxFeeTip.String(), Total: expense.Total.String(),
		Currency: expense.Currency, InvoiceURL: expense.InvoicePicUrl, AllocationMode: expense.AllocationMode, OccurredOn: occurredOn,
		Items: make([]canonicalExpenseItem, 0, len(items)), Allocations: make([]canonicalExpenseAllocation, 0, len(allocations)),
	}
	if len(receiptChoice) > 0 && receiptChoice[0] != nil {
		canonical.ReceiptKeep = receiptChoice[0].Keep
		if canonical.ReceiptKeep {
			canonical.ReceiptToken = receiptChoice[0].Token
		}
	}
	for _, item := range items {
		canonical.Items = append(canonical.Items, canonicalExpenseItem{
			Description: item.Description,
			Quantity:    decimalString(item.Quantity),
			Unit:        item.Unit,
			UnitPrice:   decimalString(item.UnitPrice),
			LineTotal:   item.LineTotal.String(),
			Position:    item.Position,
		})
	}
	for _, allocation := range allocations {
		amount := ""
		if allocation.Amount != nil {
			amount = allocation.Amount.String()
		}
		canonical.Allocations = append(canonical.Allocations, canonicalExpenseAllocation{
			UserID: allocation.UserID.String(), Amount: amount, PercentageBasisPoints: allocation.PercentageBasisPoints,
		})
	}
	slices.SortFunc(canonical.Allocations, func(a, b canonicalExpenseAllocation) int {
		return strings.Compare(a.UserID, b.UserID)
	})
	payload, err := json.Marshal(canonical)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(payload)
	return hash[:], nil
}

func decimalString(value *decimal.Decimal) *string {
	if value == nil {
		return nil
	}
	formatted := value.String()
	return &formatted
}
