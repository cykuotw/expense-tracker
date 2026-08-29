package store_test

import (
	"errors"
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"sync"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newIdempotencyExpense() types.Expense {
	return types.Expense{
		ID: uuid.New(), Description: "idempotency test", GroupID: uuid.New(), CreateByUserID: uuid.New(),
		PayByUserId: uuid.New(), ExpenseTypeID: uuid.New(), SubTotal: decimal.NewFromInt(10),
		TaxFeeTip: decimal.Zero, Total: decimal.NewFromInt(10), Currency: "CAD", SplitRule: "Equally",
	}
}

func TestExpenseCreateIdempotencyClaimReplaysOneCommittedExpense(t *testing.T) {
	db := openTestDB(t)
	store := expense.NewStore(db)
	createdID := uuid.New()
	created := types.Expense{
		ID: createdID, Description: "idempotency replay", GroupID: uuid.New(), CreateByUserID: uuid.New(),
		PayByUserId: uuid.New(), ExpenseTypeID: uuid.New(), SubTotal: decimal.NewFromInt(10),
		TaxFeeTip: decimal.Zero, Total: decimal.NewFromInt(10), Currency: "CAD", SplitRule: "Equally",
	}
	record := types.ExpenseCreateIdempotency{
		CreatorUserID: created.CreateByUserID, Key: uuid.New(), RequestFingerprint: []byte("fingerprint"), ExpenseID: createdID,
	}
	require.NoError(t, ensureExpenseParents(db, created))
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM expense_create_idempotency WHERE creator_user_id = $1 AND idempotency_key = $2", record.CreatorUserID, record.Key)
		deleteExpense(db, createdID)
	})

	require.NoError(t, store.RunInTransaction(func(tx types.ExpenseStore) error {
		_, claimed, err := tx.ClaimExpenseCreateIdempotency(record)
		if err != nil || !claimed {
			return err
		}
		return tx.CreateExpense(created)
	}))

	replayID := uuid.Nil
	require.NoError(t, store.RunInTransaction(func(tx types.ExpenseStore) error {
		existing, claimed, err := tx.ClaimExpenseCreateIdempotency(record)
		if err != nil {
			return err
		}
		assert.False(t, claimed)
		replayID = existing.ExpenseID
		return nil
	}))
	assert.Equal(t, createdID, replayID)

	var expenseCount int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM expense WHERE id = $1", createdID).Scan(&expenseCount))
	assert.Equal(t, 1, expenseCount)
}

func TestExpenseCreateIdempotencyRollbackAllowsRetry(t *testing.T) {
	db := openTestDB(t)
	store := expense.NewStore(db)
	created := newIdempotencyExpense()
	record := types.ExpenseCreateIdempotency{CreatorUserID: created.CreateByUserID, Key: uuid.New(), RequestFingerprint: []byte("rollback"), ExpenseID: created.ID}
	require.NoError(t, ensureExpenseParents(db, created))
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM expense_create_idempotency WHERE creator_user_id = $1 AND idempotency_key = $2", record.CreatorUserID, record.Key)
		deleteExpense(db, created.ID)
	})

	require.Error(t, store.RunInTransaction(func(tx types.ExpenseStore) error {
		_, claimed, err := tx.ClaimExpenseCreateIdempotency(record)
		if err != nil {
			return err
		}
		if !claimed {
			return errors.New("claim was unexpectedly replayed")
		}
		return errors.New("force rollback")
	}))
	require.NoError(t, store.RunInTransaction(func(tx types.ExpenseStore) error {
		_, claimed, err := tx.ClaimExpenseCreateIdempotency(record)
		if err != nil {
			return err
		}
		if !claimed {
			return errors.New("claim was not released by rollback")
		}
		return tx.CreateExpense(created)
	}))
}

func TestExpenseCreateIdempotencyAllowsSameKeyForDifferentUsers(t *testing.T) {
	db := openTestDB(t)
	store := expense.NewStore(db)
	key := uuid.New()
	first, second := newIdempotencyExpense(), newIdempotencyExpense()
	for _, created := range []types.Expense{first, second} {
		require.NoError(t, ensureExpenseParents(db, created))
	}
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM expense_create_idempotency WHERE idempotency_key = $1", key)
		deleteExpense(db, first.ID)
		deleteExpense(db, second.ID)
	})
	for _, created := range []types.Expense{first, second} {
		record := types.ExpenseCreateIdempotency{CreatorUserID: created.CreateByUserID, Key: key, RequestFingerprint: []byte("same-key"), ExpenseID: created.ID}
		require.NoError(t, store.RunInTransaction(func(tx types.ExpenseStore) error {
			_, claimed, err := tx.ClaimExpenseCreateIdempotency(record)
			if err != nil {
				return err
			}
			if !claimed {
				return errors.New("key collided across users")
			}
			return tx.CreateExpense(created)
		}))
	}
}

func TestExpenseCreateIdempotencyConcurrentClaimsCreateOnceAndSurviveSoftDelete(t *testing.T) {
	db := openTestDB(t)
	store := expense.NewStore(db)
	created := newIdempotencyExpense()
	record := types.ExpenseCreateIdempotency{CreatorUserID: created.CreateByUserID, Key: uuid.New(), RequestFingerprint: []byte("concurrent"), ExpenseID: created.ID}
	require.NoError(t, ensureExpenseParents(db, created))
	t.Cleanup(func() {
		_, _ = db.Exec("DELETE FROM expense_create_idempotency WHERE creator_user_id = $1 AND idempotency_key = $2", record.CreatorUserID, record.Key)
		deleteExpense(db, created.ID)
	})
	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wg.Go(func() {
			errs <- store.RunInTransaction(func(tx types.ExpenseStore) error {
				_, claimed, err := tx.ClaimExpenseCreateIdempotency(record)
				if err != nil {
					return err
				}
				if !claimed {
					return nil
				}
				return tx.CreateExpense(created)
			})
		})
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		require.NoError(t, err)
	}
	var count int
	require.NoError(t, db.QueryRow("SELECT COUNT(*) FROM expense WHERE id = $1", created.ID).Scan(&count))
	assert.Equal(t, 1, count)
	_, err := db.Exec("UPDATE expense SET is_deleted = TRUE WHERE id = $1", created.ID)
	require.NoError(t, err)
	require.NoError(t, store.RunInTransaction(func(tx types.ExpenseStore) error {
		existing, claimed, err := tx.ClaimExpenseCreateIdempotency(record)
		if err != nil {
			return err
		}
		assert.False(t, claimed)
		assert.Equal(t, created.ID, existing.ExpenseID)
		return nil
	}))
}
