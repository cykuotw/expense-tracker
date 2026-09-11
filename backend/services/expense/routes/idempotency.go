package expense

import (
	"cmp"
	"crypto/sha256"
	"encoding/json"
	"slices"

	"expense-tracker/backend/types"
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
}

type canonicalExpenseItem struct {
	Name      string `json:"name"`
	Amount    string `json:"amount"`
	Unit      string `json:"unit"`
	UnitPrice string `json:"unitPrice"`
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
	for _, item := range items {
		canonical.Items = append(canonical.Items, canonicalExpenseItem{Name: item.Name, Amount: item.Amount.String(), Unit: item.Unit, UnitPrice: item.UnitPrice.String()})
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
	slices.SortFunc(canonical.Items, func(a, b canonicalExpenseItem) int {
		return cmp.Or(
			cmp.Compare(a.Name, b.Name),
			cmp.Compare(a.Amount, b.Amount),
			cmp.Compare(a.Unit, b.Unit),
			cmp.Compare(a.UnitPrice, b.UnitPrice),
		)
	})
	slices.SortFunc(canonical.Allocations, func(a, b canonicalExpenseAllocation) int {
		return cmp.Compare(a.UserID, b.UserID)
	})
	payload, err := json.Marshal(canonical)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(payload)
	return hash[:], nil
}
