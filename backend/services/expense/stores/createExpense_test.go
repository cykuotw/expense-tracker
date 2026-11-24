package store_test

import (
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateExpense(t *testing.T) {
	// prepare test data
	db := openTestDB(t)
	store := expense.NewStore(db)

	// define test cases
	type testcase struct {
		name        string
		mockExpense types.Expense
		expectFail  bool
		expectError error
	}

	subtests := []testcase{
		{
			name: "valid",
			mockExpense: types.Expense{
				ID:             uuid.New(),
				Description:    "test desc",
				GroupID:        mockGroupID,
				CreateByUserID: mockCreatorID,
				PayByUserId:    mockPayerID,
				ExpenseTypeID:  uuid.New(),
				CreateTime:     time.Now(),
				ProviderName:   "test prov",
				IsSettled:      false,
				SubTotal:       decimal.NewFromFloat(20.01),
				TaxFeeTip:      decimal.NewFromFloat(1.01),
				Total:          decimal.NewFromFloat(21.02),
				Currency:       "CAD",
				InvoicePicUrl:  "http://mockpic.url.com",
				AllocationMode: "exact",
				OccurredOn:     "2026-08-31",
			},
			expectFail:  false,
			expectError: nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			assert.NoError(t, ensureExpenseParents(db, test.mockExpense))
			startedAt := time.Now().UTC().Add(-time.Second)
			err := store.CreateExpense(test.mockExpense)
			finishedAt := time.Now().UTC().Add(time.Second)
			defer deleteExpense(db, test.mockExpense.ID)

			assert.Equal(t, test.expectError, err)
			created := selectExpenseByID(db, test.mockExpense.ID)
			for _, instant := range []time.Time{
				created.CreateTime,
				created.UpdateTime,
				created.ExpenseTime,
			} {
				assert.False(t, instant.IsZero())
				assert.False(t, instant.Before(startedAt))
				assert.False(t, instant.After(finishedAt))
			}
			assert.Equal(t, test.mockExpense.OccurredOn, created.OccurredOn)
		})
	}
}

func TestLoyaltyRedemptionRoundTrip(t *testing.T) {
	database := openTestDB(t)
	store := expense.NewStore(database)
	purchase := newTestExpense(uuid.New())
	purchase.SubTotal = decimal.RequireFromString("24.48")
	purchase.TaxFeeTip = decimal.RequireFromString("3.18")
	purchase.LoyaltyRedemption = new(decimal.RequireFromString("27.66"))
	purchase.Total = decimal.Zero
	purchase.OCRRequestID = new(uuid.NewString())
	purchase.ReceiptScanNote = "Partial payment with CT Money"
	require.NoError(t, ensureExpenseParents(database, purchase))
	require.NoError(t, store.CreateExpense(purchase))
	t.Cleanup(func() { deleteExpense(database, purchase.ID) })
	check := func() {
		t.Helper()
		saved, err := store.GetExpenseByID(purchase.ID.String())
		require.NoError(t, err)
		require.Equal(t, purchase.OCRRequestID, saved.OCRRequestID)
		require.Equal(t, purchase.ReceiptScanNote, saved.ReceiptScanNote)
		require.Equal(t, purchase.EstimatedScanCostUSD, saved.EstimatedScanCostUSD)
		require.NotNil(t, saved.LoyaltyRedemption)
		require.True(t, saved.LoyaltyRedemption.Equal(*purchase.LoyaltyRedemption))
		require.True(t, saved.Total.Equal(purchase.Total))
		require.Equal(t, "27.66", saved.Total.Add(*saved.LoyaltyRedemption).String())
	}
	check()
	_, err := database.Exec(`INSERT INTO ocr_usage
        (request_id, account_id, model, outcome, usage_status, cost_status, pricing_version, estimated_cost_usd,
         input_tokens, output_tokens, thinking_tokens, cached_input_tokens, tool_input_tokens, total_tokens, http_status, latency_ms, recorded_at)
        VALUES ($1, $2, 'test-model', 'success', 'available', 'estimated', 'test', 0.000001234,
                1, 1, 0, 0, 0, 2, 200, 1, NOW())`, *purchase.OCRRequestID, purchase.CreateByUserID)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = database.Exec("DELETE FROM ocr_usage WHERE request_id = $1", *purchase.OCRRequestID) })
	purchase.EstimatedScanCostUSD = new(decimal.RequireFromString("0.000001234"))
	purchase.LoyaltyRedemption = new(decimal.NewFromInt(10))
	purchase.Total = decimal.RequireFromString("17.66")
	require.NoError(t, store.UpdateExpense(purchase))
	check()
}
