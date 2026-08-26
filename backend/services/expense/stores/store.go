package store

import (
	"database/sql"
	"errors"
	"expense-tracker/backend/types"
)

type queryExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
}

type transaction interface {
	queryExecutor
	Commit() error
	Rollback() error
}

type Store struct {
	db      queryExecutor
	beginTx func() (transaction, error)
}

func NewStore(db *sql.DB) *Store {
	return &Store{
		db: db,
		beginTx: func() (transaction, error) {
			return db.Begin()
		},
	}
}

func (s *Store) RunInTransaction(callback func(types.ExpenseStore) error) error {
	if s.beginTx == nil {
		return errors.New("transactions are unavailable on a transaction-bound expense store")
	}

	tx, err := s.beginTx()
	if err != nil {
		return err
	}

	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := callback(&Store{db: tx}); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return err
	}

	committed = true
	return nil
}
