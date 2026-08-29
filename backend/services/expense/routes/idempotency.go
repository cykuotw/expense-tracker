package expense

import (
	"crypto/sha256"
	"encoding/json"
	"sort"

	"expense-tracker/backend/types"
)

type canonicalExpenseCreate struct {
	Description   string                   `json:"description"`
	GroupID       string                   `json:"groupId"`
	PayerID       string                   `json:"payByUserId"`
	ExpenseTypeID string                   `json:"expenseTypeId"`
	ProviderName  string                   `json:"providerName"`
	SubTotal      string                   `json:"subTotal"`
	TaxFeeTip     string                   `json:"taxFeeTip"`
	Total         string                   `json:"total"`
	Currency      string                   `json:"currency"`
	InvoiceURL    string                   `json:"invoiceUrl"`
	SplitRule     string                   `json:"splitRule"`
	Items         []canonicalExpenseItem   `json:"items"`
	Ledgers       []canonicalExpenseLedger `json:"ledgers"`
}

type canonicalExpenseItem struct {
	Name      string `json:"name"`
	Amount    string `json:"amount"`
	Unit      string `json:"unit"`
	UnitPrice string `json:"unitPrice"`
}

type canonicalExpenseLedger struct {
	LenderID   string `json:"lenderUserId"`
	BorrowerID string `json:"borrowerUserId"`
	Share      string `json:"share"`
}

func expenseCreateFingerprint(expense types.Expense, items []types.Item, ledgers []types.Ledger) ([]byte, error) {
	canonical := canonicalExpenseCreate{
		Description: expense.Description, GroupID: expense.GroupID.String(), PayerID: expense.PayByUserId.String(),
		ExpenseTypeID: expense.ExpenseTypeID.String(), ProviderName: expense.ProviderName,
		SubTotal: expense.SubTotal.String(), TaxFeeTip: expense.TaxFeeTip.String(), Total: expense.Total.String(),
		Currency: expense.Currency, InvoiceURL: expense.InvoicePicUrl, SplitRule: expense.SplitRule,
		Items: make([]canonicalExpenseItem, 0, len(items)), Ledgers: make([]canonicalExpenseLedger, 0, len(ledgers)),
	}
	for _, item := range items {
		canonical.Items = append(canonical.Items, canonicalExpenseItem{Name: item.Name, Amount: item.Amount.String(), Unit: item.Unit, UnitPrice: item.UnitPrice.String()})
	}
	for _, ledger := range ledgers {
		canonical.Ledgers = append(canonical.Ledgers, canonicalExpenseLedger{LenderID: ledger.LenderUserID.String(), BorrowerID: ledger.BorrowerUesrID.String(), Share: ledger.Share.String()})
	}
	sort.Slice(canonical.Items, func(i, j int) bool {
		return canonical.Items[i].Name+canonical.Items[i].Amount+canonical.Items[i].Unit+canonical.Items[i].UnitPrice < canonical.Items[j].Name+canonical.Items[j].Amount+canonical.Items[j].Unit+canonical.Items[j].UnitPrice
	})
	sort.Slice(canonical.Ledgers, func(i, j int) bool {
		return canonical.Ledgers[i].LenderID+canonical.Ledgers[i].BorrowerID+canonical.Ledgers[i].Share < canonical.Ledgers[j].LenderID+canonical.Ledgers[j].BorrowerID+canonical.Ledgers[j].Share
	})
	payload, err := json.Marshal(canonical)
	if err != nil {
		return nil, err
	}
	hash := sha256.Sum256(payload)
	return hash[:], nil
}
