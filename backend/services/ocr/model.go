package ocr

import (
	"context"
	"regexp"
	"strings"

	"github.com/shopspring/decimal"
)

type Provenance string

const (
	ProvenanceProvider Provenance = "provider"
	ProvenanceInferred Provenance = "inferred"
)

// Field is a normalized provider value and its provider-reported confidence.
type Field struct {
	Value          string     `json:"value"`
	Confidence     float32    `json:"confidence,omitempty"`
	Provenance     Provenance `json:"provenance,omitempty"`
	RequiresReview bool       `json:"requiresReview,omitempty"`
}

// Item is one normalized receipt line. Blank optional fields were not detected.
type Item struct {
	Description Field `json:"description"`
	Quantity    Field `json:"quantity"`
	UnitPrice   Field `json:"unitPrice"`
	LineTotal   Field `json:"lineTotal"`
	RawRow      Field `json:"-"`
}

// Result is provider-neutral OCR output suitable for an editable expense draft.
type Result struct {
	Merchant Field  `json:"merchant"`
	Date     Field  `json:"date"`
	Currency Field  `json:"currencySuggestion"`
	Subtotal Field  `json:"subtotal"`
	Tax      Field  `json:"tax"`
	Tip      Field  `json:"tip"`
	Total    Field  `json:"total"`
	Items    []Item `json:"items"`
}

// Provider analyzes one already-normalized document image.
type Provider interface {
	AnalyzeExpense(ctx context.Context, document []byte) (Result, error)
}

var unambiguousMoney = regexp.MustCompile(`^[+-]?[0-9]+(?:\.[0-9]+)?$`)

// PrepareDraft adds application-level review and inference semantics without
// mutating the provider-neutral extraction values.
func PrepareDraft(result Result) Result {
	if result.Items == nil {
		result.Items = []Item{}
	}
	markProviderFields(&result)
	result.Merchant.RequiresReview = result.Merchant.Value != ""
	result.Date.RequiresReview = result.Date.Value != ""
	result.Currency.RequiresReview = result.Currency.Value != ""

	if result.Tax.Value == "" && sameMoney(result.Subtotal.Value, result.Total.Value) {
		result.Tax = Field{Value: "0", Provenance: ProvenanceInferred, RequiresReview: true}
	}
	for index := range result.Items {
		item := &result.Items[index]
		if item.Quantity.Value == "" && sameMoney(item.UnitPrice.Value, item.LineTotal.Value) {
			item.Quantity = Field{Value: "1", Provenance: ProvenanceInferred, RequiresReview: true}
		}
	}
	return result
}

func markProviderFields(result *Result) {
	fields := []*Field{
		&result.Merchant, &result.Date, &result.Currency, &result.Subtotal,
		&result.Tax, &result.Tip, &result.Total,
	}
	for index := range result.Items {
		item := &result.Items[index]
		fields = append(fields, &item.Description, &item.Quantity, &item.UnitPrice, &item.LineTotal)
	}
	for _, field := range fields {
		if field.Value != "" {
			field.Provenance = ProvenanceProvider
		}
	}
}

func sameMoney(left, right string) bool {
	leftValue, leftOK := parseUnambiguousMoney(left)
	rightValue, rightOK := parseUnambiguousMoney(right)
	return leftOK && rightOK && leftValue.Equal(rightValue)
}

func parseUnambiguousMoney(value string) (decimal.Decimal, bool) {
	normalized := strings.TrimSpace(value)
	normalized = strings.TrimSpace(strings.TrimPrefix(normalized, "$"))
	normalized = strings.TrimSpace(strings.TrimSuffix(strings.ToUpper(normalized), "CAD"))
	normalized = strings.ReplaceAll(normalized, ",", "")
	if !unambiguousMoney.MatchString(normalized) {
		return decimal.Decimal{}, false
	}
	parsed, err := decimal.NewFromString(normalized)
	return parsed, err == nil
}
