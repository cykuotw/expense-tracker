package expense

import (
	"testing"

	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNormalizeCreateItemsSupportsConfirmedAndLegacyContracts(t *testing.T) {
	expenseID := uuid.New()
	quantity := decimal.RequireFromString("2.5")
	unitPrice := decimal.RequireFromString("3.125")
	lineTotal := decimal.RequireFromString("7.50")
	unit := " serving "
	legacyAmount := decimal.NewFromInt(2)
	legacyPrice := decimal.RequireFromString("4.25")
	legacyUnit := " each "

	items, err := normalizeCreateItems(expenseID, []types.ItemPayload{
		{Description: "  Lunch  ", Quantity: &quantity, Unit: &unit, UnitPrice: &unitPrice, LineTotal: &lineTotal},
		{ItemName: "Legacy", Amount: &legacyAmount, Unit: &legacyUnit, UnitPrice: &legacyPrice},
	})

	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "Lunch", items[0].Description)
	assert.Equal(t, "serving", *items[0].Unit)
	assert.Equal(t, "7.5", items[0].LineTotal.String())
	assert.Equal(t, int32(0), items[0].Position)
	assert.Equal(t, "8.5", items[1].LineTotal.String())
	assert.Equal(t, int32(1), items[1].Position)
}

func TestNormalizeItemsRejectsAmbiguousOrUnconfirmedValues(t *testing.T) {
	value := decimal.NewFromInt(1)
	_, err := normalizeCreateItems(uuid.New(), []types.ItemPayload{{Description: "New", ItemName: "Legacy", LineTotal: &value}})
	require.ErrorIs(t, err, types.ErrInvalidMoney)

	_, err = normalizeCreateItems(uuid.New(), []types.ItemPayload{{Description: "Missing total"}})
	require.ErrorIs(t, err, types.ErrInvalidMoney)
}

func TestNormalizeUpdateItemsRejectsDuplicateIDs(t *testing.T) {
	itemID := uuid.New()
	total := decimal.NewFromInt(1)
	_, err := normalizeUpdateItems(uuid.New(), []types.ItemUpdatePayload{
		{ID: itemID, ItemPayload: types.ItemPayload{Description: "One", LineTotal: &total}},
		{ID: itemID, ItemPayload: types.ItemPayload{Description: "Two", LineTotal: &total}},
	})
	require.ErrorIs(t, err, types.ErrInvalidMoney)
}
