package ocr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrepareDraftAppliesConservativeInferenceAndProvenance(t *testing.T) {
	draft := PrepareDraft(Result{
		Merchant: Field{Value: "Example Market", Confidence: 91},
		Date:     Field{Value: "2026-09-25", Confidence: 82},
		Currency: Field{Value: "CAD", Confidence: 77},
		Subtotal: Field{Value: "$12.00", Confidence: 95},
		Total:    Field{Value: "12.00 CAD", Confidence: 96},
		Items: []Item{{
			Description: Field{Value: "Milk", Confidence: 98},
			UnitPrice:   Field{Value: "$4.00", Confidence: 90},
			LineTotal:   Field{Value: "4.00", Confidence: 97},
		}},
	})

	assert.Equal(t, ProvenanceProvider, draft.Merchant.Provenance)
	assert.Equal(t, "12", draft.Subtotal.Value)
	assert.Equal(t, "12", draft.Total.Value)
	assert.Equal(t, "4", draft.Items[0].UnitPrice.Value)
	assert.True(t, draft.Merchant.RequiresReview)
	assert.True(t, draft.Date.RequiresReview)
	assert.True(t, draft.Currency.RequiresReview)
	assert.Equal(t, Field{Value: "0", Provenance: ProvenanceInferred, RequiresReview: true}, draft.Tax)
	assert.Equal(t, Field{Value: "1", Provenance: ProvenanceInferred, RequiresReview: true}, draft.Items[0].Quantity)
	assert.Equal(t, ProvenanceProvider, draft.Items[0].LineTotal.Provenance)
}

func TestPrepareDraftNormalizesUnambiguousProviderValues(t *testing.T) {
	draft := PrepareDraft(Result{
		Date:     Field{Value: "Jan 18, 2022", Confidence: 82},
		Subtotal: Field{Value: "$1,024.50", Confidence: 93},
		Tax:      Field{Value: "2.32 CAD", Confidence: 94},
		Total:    Field{Value: "$1,026.82", Confidence: 99},
	})

	assert.Equal(t, "2022-01-18", draft.Date.Value)
	assert.Equal(t, "1024.5", draft.Subtotal.Value)
	assert.Equal(t, "2.32", draft.Tax.Value)
	assert.Equal(t, "1026.82", draft.Total.Value)
	assert.Equal(t, float32(99), draft.Total.Confidence)
	assert.Equal(t, ProvenanceProvider, draft.Total.Provenance)
}

func TestPrepareDraftPreservesAmbiguousProviderDate(t *testing.T) {
	draft := PrepareDraft(Result{Date: Field{Value: "01/02/2022", Confidence: 75}})

	assert.Equal(t, "01/02/2022", draft.Date.Value)
	assert.Equal(t, float32(75), draft.Date.Confidence)
	assert.Equal(t, ProvenanceProvider, draft.Date.Provenance)
}

func TestPrepareDraftDoesNotInferFromAmbiguousOrUnequalMoney(t *testing.T) {
	draft := PrepareDraft(Result{
		Subtotal: Field{Value: "10,00"},
		Total:    Field{Value: "10.00"},
		Items: []Item{{
			UnitPrice: Field{Value: "2 x 4.00"},
			LineTotal: Field{Value: "8.00"},
		}},
	})

	assert.Empty(t, draft.Tax.Value)
	assert.Equal(t, "10,00", draft.Subtotal.Value)
	assert.Empty(t, draft.Items[0].Quantity.Value)
	assert.Empty(t, draft.Tip.Value)
}
