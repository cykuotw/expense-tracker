package ocr

import "context"

// Field is a normalized provider value and its provider-reported confidence.
type Field struct {
	Value      string
	Confidence float32
}

// Item is one normalized receipt line. Blank optional fields were not detected.
type Item struct {
	Description Field
	Quantity    Field
	UnitPrice   Field
	LineTotal   Field
	RawRow      Field
}

// Result is provider-neutral OCR output suitable for an editable expense draft.
type Result struct {
	Merchant Field
	Date     Field
	Currency Field
	Subtotal Field
	Tax      Field
	Tip      Field
	Total    Field
	Items    []Item
}

// Provider analyzes one already-normalized document image.
type Provider interface {
	AnalyzeExpense(ctx context.Context, document []byte) (Result, error)
}
