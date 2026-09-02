package store

import (
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type recordedQuery struct {
	statement string
	args      []any
}

type recordingExecutor struct {
	execCalls  []recordedQuery
	queryCalls []recordedQuery
	queryErr   error
}

func (e *recordingExecutor) Exec(query string, args ...any) (sql.Result, error) {
	e.execCalls = append(e.execCalls, recordedQuery{statement: query, args: args})
	return recordingResult(1), nil
}

func (e *recordingExecutor) Query(query string, args ...any) (*sql.Rows, error) {
	e.queryCalls = append(e.queryCalls, recordedQuery{statement: query, args: args})
	return nil, e.queryErr
}

type recordingResult int64

func (r recordingResult) LastInsertId() (int64, error) { return 0, nil }
func (r recordingResult) RowsAffected() (int64, error) { return int64(r), nil }

func TestExpenseReadsUseExplicitColumnProjection(t *testing.T) {
	tests := []struct {
		name string
		read func(*Store) error
	}{
		{
			name: "detail",
			read: func(store *Store) error {
				_, err := store.GetExpenseByID(uuid.NewString())
				return err
			},
		},
		{
			name: "list",
			read: func(store *Store) error {
				_, err := store.GetExpenseList(uuid.NewString(), 0, types.ExpenseListOrderNewest, types.ExpenseListStatusAll)
				return err
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			queryErr := errors.New("stop after recording query")
			executor := &recordingExecutor{queryErr: queryErr}
			store := &Store{db: executor}

			err := test.read(store)

			require.ErrorIs(t, err, queryErr)
			require.Len(t, executor.queryCalls, 1)
			query := executor.queryCalls[0].statement
			assert.NotContains(t, strings.ToUpper(query), "SELECT *")
			for _, column := range []string{
				"id", "description", "create_time_utc", "update_time_utc",
				"expense_time_utc", "delete_time_utc", "settle_time_utc", "occurred_on",
			} {
				assert.Contains(t, query, column)
			}
		})
	}
}

func TestExpenseListOrdersByOccurrenceDateWithLegacyFallback(t *testing.T) {
	for _, order := range []types.ExpenseListOrder{types.ExpenseListOrderNewest, types.ExpenseListOrderOldest} {
		t.Run(string(order), func(t *testing.T) {
			queryErr := errors.New("stop after recording query")
			executor := &recordingExecutor{queryErr: queryErr}
			store := &Store{db: executor}

			_, err := store.GetExpenseList(uuid.NewString(), 0, order, types.ExpenseListStatusAll)

			require.ErrorIs(t, err, queryErr)
			require.Len(t, executor.queryCalls, 1)
			query := executor.queryCalls[0].statement
			assert.Contains(t, query, "COALESCE(occurred_on, (expense_time_utc AT TIME ZONE 'UTC')::date)")
			assert.Contains(t, query, "expense_time_utc")
		})
	}
}

func TestCreateExpenseWritesUTCInstantsAsTimeValues(t *testing.T) {
	executor := &recordingExecutor{}
	store := &Store{db: executor}
	expense := testExpense()
	startedAt := time.Now().UTC()

	require.NoError(t, store.CreateExpense(expense))

	finishedAt := time.Now().UTC()
	require.Len(t, executor.execCalls, 1)
	args := executor.execCalls[0].args
	for _, index := range []int{13, 14, 15} {
		assertUTCInstantBetween(t, args[index], startedAt, finishedAt)
	}
	assert.Equal(t, expense.OccurredOn, args[17])
}

func TestUpdateExpensePreservesLegacyOccurrenceAndWritesFreshUTCInstant(t *testing.T) {
	executor := &recordingExecutor{}
	store := &Store{db: executor}
	expense := testExpense()
	expense.UpdateTime = time.Time{}
	expense.ExpenseTime = time.Time{}
	startedAt := time.Now().UTC()

	require.NoError(t, store.UpdateExpense(expense))

	finishedAt := time.Now().UTC()
	require.Len(t, executor.execCalls, 1)
	call := executor.execCalls[0]
	assert.NotContains(t, call.statement, "expense_time_utc")
	require.Len(t, call.args, 15)
	assertUTCInstantBetween(t, call.args[3], startedAt, finishedAt)
	assert.Equal(t, expense.OccurredOn, call.args[13])
}

func TestMutationAuditWritesUseUTCInstants(t *testing.T) {
	tests := []struct {
		name        string
		expectedSQL string
		mutate      func(*Store) error
	}{
		{
			name:        "settle expenses from group settlement",
			expectedSQL: "settle_time_utc",
			mutate: func(store *Store) error {
				return store.UpdateExpenseSettleInGroup(uuid.NewString())
			},
		},
		{
			name:        "settle expenses after balances settle",
			expectedSQL: "settle_time_utc",
			mutate: func(store *Store) error {
				return store.SettleExpenseByGroupId(uuid.NewString())
			},
		},
		{
			name:        "settle balance",
			expectedSQL: "settle_time_utc",
			mutate: func(store *Store) error {
				return store.SettleBalanceByBalanceId(uuid.NewString(), uuid.NewString())
			},
		},
		{
			name:        "soft delete expense",
			expectedSQL: "delete_time_utc",
			mutate: func(store *Store) error {
				return store.DeleteExpense(types.Expense{ID: uuid.New()})
			},
		},
		{
			name:        "outdate balances",
			expectedSQL: "update_time_utc",
			mutate: func(store *Store) error {
				return store.OutdateBalanceByGroupId(uuid.NewString())
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			executor := &recordingExecutor{}
			store := &Store{db: executor}
			startedAt := time.Now().UTC()

			require.NoError(t, test.mutate(store))

			finishedAt := time.Now().UTC()
			require.Len(t, executor.execCalls, 1)
			call := executor.execCalls[0]
			assert.Contains(t, call.statement, test.expectedSQL)
			require.NotEmpty(t, call.args)
			assertUTCInstantBetween(t, call.args[0], startedAt, finishedAt)
		})
	}
}

func assertUTCInstantBetween(t *testing.T, value any, startedAt, finishedAt time.Time) {
	t.Helper()

	instant, ok := value.(time.Time)
	require.True(t, ok, "expected time.Time, got %T", value)
	assert.False(t, instant.IsZero())
	assert.Equal(t, time.UTC, instant.Location())
	assert.False(t, instant.Before(startedAt))
	assert.False(t, instant.After(finishedAt))
}

func testExpense() types.Expense {
	return types.Expense{
		ID:             uuid.New(),
		Description:    "test expense",
		GroupID:        uuid.New(),
		CreateByUserID: uuid.New(),
		PayByUserId:    uuid.New(),
		ExpenseTypeID:  uuid.New(),
		SubTotal:       decimal.NewFromInt(10),
		TaxFeeTip:      decimal.NewFromInt(1),
		Total:          decimal.NewFromInt(11),
		Currency:       "CAD",
		SplitRule:      "Equally",
		OccurredOn:     "2026-09-01",
	}
}
