package store

import (
	"database/sql"
	"errors"
	"expense-tracker/backend/types"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type fakeTransaction struct {
	commitErr     error
	rollbackErr   error
	commitCalls   int
	rollbackCalls int
}

func (tx *fakeTransaction) Exec(string, ...any) (sql.Result, error) {
	return nil, nil
}

func (tx *fakeTransaction) Query(string, ...any) (*sql.Rows, error) {
	return nil, nil
}

func (tx *fakeTransaction) Commit() error {
	tx.commitCalls++
	return tx.commitErr
}

func (tx *fakeTransaction) Rollback() error {
	tx.rollbackCalls++
	return tx.rollbackErr
}

func TestRunInTransactionCommitsSuccessfulCallback(t *testing.T) {
	tx := &fakeTransaction{}
	store := &Store{
		beginTx: func() (transaction, error) {
			return tx, nil
		},
	}

	callbackCalls := 0
	err := store.RunInTransaction(func(transactionStore types.ExpenseStore) error {
		callbackCalls++
		boundStore, ok := transactionStore.(*Store)
		require.True(t, ok)
		assert.Same(t, tx, boundStore.db)
		return nil
	})

	require.NoError(t, err)
	assert.Equal(t, 1, callbackCalls)
	assert.Equal(t, 1, tx.commitCalls)
	assert.Zero(t, tx.rollbackCalls)
}

func TestRunInTransactionReturnsBeginError(t *testing.T) {
	beginErr := errors.New("begin failed")
	store := &Store{
		beginTx: func() (transaction, error) {
			return nil, beginErr
		},
	}

	callbackCalled := false
	err := store.RunInTransaction(func(types.ExpenseStore) error {
		callbackCalled = true
		return nil
	})

	assert.ErrorIs(t, err, beginErr)
	assert.False(t, callbackCalled)
}

func TestRunInTransactionRollsBackCallbackError(t *testing.T) {
	callbackErr := errors.New("callback failed")
	tx := &fakeTransaction{}
	store := &Store{
		beginTx: func() (transaction, error) {
			return tx, nil
		},
	}

	err := store.RunInTransaction(func(types.ExpenseStore) error {
		return callbackErr
	})

	assert.ErrorIs(t, err, callbackErr)
	assert.Zero(t, tx.commitCalls)
	assert.Equal(t, 1, tx.rollbackCalls)
}

func TestRunInTransactionRollsBackCommitError(t *testing.T) {
	commitErr := errors.New("commit failed")
	tx := &fakeTransaction{commitErr: commitErr}
	store := &Store{
		beginTx: func() (transaction, error) {
			return tx, nil
		},
	}

	err := store.RunInTransaction(func(types.ExpenseStore) error {
		return nil
	})

	assert.ErrorIs(t, err, commitErr)
	assert.Equal(t, 1, tx.commitCalls)
	assert.Equal(t, 1, tx.rollbackCalls)
}
