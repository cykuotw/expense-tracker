package store

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"testing"

	"expense-tracker/backend/internal/testsql"
	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type itemExecCall struct {
	query string
	args  []any
}

type itemRecordingDB struct {
	calls []itemExecCall
}

func (db *itemRecordingDB) Exec(query string, args ...any) (sql.Result, error) {
	db.calls = append(db.calls, itemExecCall{query: query, args: args})
	return itemResult(1), nil
}

func (*itemRecordingDB) Query(string, ...any) (*sql.Rows, error) {
	return nil, errors.New("unexpected query")
}

type itemResult int64

func (itemResult) LastInsertId() (int64, error)   { return 0, nil }
func (r itemResult) RowsAffected() (int64, error) { return int64(r), nil }

func TestCreateItemDualWritesLegacyAndConfirmedContracts(t *testing.T) {
	database := &itemRecordingDB{}
	store := &Store{db: database}
	quantity := decimal.RequireFromString("2.5")
	unitPrice := decimal.RequireFromString("3.125")
	unit := "serving"
	item := types.Item{
		ID: uuid.New(), ExpenseID: uuid.New(), Description: "Lunch",
		Quantity: &quantity, Unit: &unit, UnitPrice: &unitPrice,
		LineTotal: decimal.RequireFromString("7.50"), Position: 2,
	}

	require.NoError(t, store.CreateItem(item))
	require.Len(t, database.calls, 1)
	call := database.calls[0]
	assert.Contains(t, call.query, "description, quantity, confirmed_unit, confirmed_unit_price, line_total, position")
	assert.Equal(t, "Lunch", call.args[2])
	assert.Equal(t, "1", call.args[3])
	assert.Equal(t, "serving", call.args[4])
	assert.Equal(t, "7.5", call.args[5])
	assert.Equal(t, item.Description, call.args[6])
	assert.Equal(t, item.Position, call.args[11])
}

func TestGetItemsResolvesLegacyAndConfirmedRows(t *testing.T) {
	expenseID := uuid.New()
	legacyID := uuid.New()
	confirmedID := uuid.New()
	database, cleanup := testsql.Open(testsql.Result{
		Columns: []string{"id", "expense_id", "description", "quantity", "unit", "unit_price", "line_total", "position"},
		Rows: [][]driver.Value{
			{legacyID.String(), expenseID.String(), "Legacy", "2.000000", "each", "3.000", "6.000", int64(0)},
			{confirmedID.String(), expenseID.String(), "Confirmed", nil, nil, nil, "4.500", int64(1)},
		},
	})
	defer cleanup()

	items, err := NewStore(database).GetItemsByExpenseID(expenseID.String())

	require.NoError(t, err)
	require.Len(t, items, 2)
	assert.Equal(t, "2", items[0].Quantity.String())
	assert.Equal(t, "each", *items[0].Unit)
	assert.Equal(t, "3", items[0].UnitPrice.String())
	assert.Equal(t, "6", items[0].LineTotal.String())
	assert.Nil(t, items[1].Quantity)
	assert.Nil(t, items[1].Unit)
	assert.Nil(t, items[1].UnitPrice)
	assert.Equal(t, int32(1), items[1].Position)
}

func TestItemReconciliationScopesDeleteAndClearsPositions(t *testing.T) {
	database := &itemRecordingDB{}
	store := &Store{db: database}
	expenseID := uuid.New()
	itemIDs := []uuid.UUID{uuid.New(), uuid.New()}

	require.NoError(t, store.ClearItemPositions(expenseID))
	require.NoError(t, store.DeleteItemsNotIn(expenseID, itemIDs))
	require.NoError(t, store.DeleteItemsNotIn(expenseID, nil))

	require.Len(t, database.calls, 3)
	assert.Contains(t, database.calls[0].query, "SET position = NULL WHERE expense_id = $1")
	assert.Contains(t, database.calls[1].query, "expense_id = $1 AND id NOT IN ($2, $3)")
	assert.Equal(t, []any{expenseID, itemIDs[0], itemIDs[1]}, database.calls[1].args)
	assert.Equal(t, "DELETE FROM item WHERE expense_id = $1", database.calls[2].query)
}
