package store

import (
	"database/sql"
	"errors"

	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

func (s *Store) CreateExpenseAllocation(allocation types.ExpenseAllocation) error {
	var amount any
	if allocation.Amount != nil {
		amount = allocation.Amount.String()
	}
	var percentageBasisPoints any
	if allocation.PercentageBasisPoints != nil {
		percentageBasisPoints = *allocation.PercentageBasisPoints
	}

	_, err := s.db.Exec(
		`INSERT INTO expense_allocation (
			expense_id, user_id, amount, percentage_basis_points
		) VALUES ($1, $2, $3, $4);`,
		allocation.ExpenseID,
		allocation.UserID,
		amount,
		percentageBasisPoints,
	)
	return err
}

func (s *Store) GetExpenseAllocationsByExpenseID(expenseID string) ([]types.ExpenseAllocation, error) {
	rows, err := s.db.Query(
		`SELECT expense_id, user_id, amount, percentage_basis_points
		FROM expense_allocation
		WHERE expense_id = $1
		ORDER BY user_id ASC;`,
		expenseID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allocations := make([]types.ExpenseAllocation, 0)
	for rows.Next() {
		var allocation types.ExpenseAllocation
		var amount sql.NullString
		var percentageBasisPoints sql.NullInt32
		if err := rows.Scan(
			&allocation.ExpenseID,
			&allocation.UserID,
			&amount,
			&percentageBasisPoints,
		); err != nil {
			return nil, err
		}
		if amount.Valid {
			parsed, err := decimal.NewFromString(amount.String)
			if err != nil {
				return nil, err
			}
			allocation.Amount = &parsed
		}
		if percentageBasisPoints.Valid {
			value := percentageBasisPoints.Int32
			allocation.PercentageBasisPoints = &value
		}
		allocations = append(allocations, allocation)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	if len(allocations) == 0 {
		return nil, types.ErrExpenseNotExist
	}
	return allocations, nil
}

var errAllocationTransactionRequired = errors.New("expense allocation reconciliation requires a transaction-bound store")

func (s *Store) ReconcileExpenseAllocationState(
	expenseID, payerID uuid.UUID,
	allocations []types.ExpenseAllocation,
	ledgers []types.Ledger,
) error {
	if !s.transactionBound {
		return errAllocationTransactionRequired
	}
	if err := s.replaceExpenseAllocations(expenseID, allocations); err != nil {
		return err
	}
	return s.reconcileExpenseLedgers(expenseID, payerID, ledgers)
}

func (s *Store) replaceExpenseAllocations(expenseID uuid.UUID, allocations []types.ExpenseAllocation) error {
	seen := make(map[uuid.UUID]struct{}, len(allocations))
	for _, allocation := range allocations {
		if allocation.ExpenseID != expenseID {
			return types.ErrInvalidAction
		}
		if _, exists := seen[allocation.UserID]; exists {
			return types.ErrInvalidAction
		}
		seen[allocation.UserID] = struct{}{}
	}
	if _, err := s.db.Exec("DELETE FROM expense_allocation WHERE expense_id = $1;", expenseID); err != nil {
		return err
	}
	for _, allocation := range allocations {
		if err := s.CreateExpenseAllocation(allocation); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) reconcileExpenseLedgers(expenseID, payerID uuid.UUID, ledgers []types.Ledger) error {
	requestedBorrowers := make(map[uuid.UUID]struct{}, len(ledgers))
	for _, ledger := range ledgers {
		if _, exists := requestedBorrowers[ledger.BorrowerUesrID]; exists {
			return types.ErrInvalidAction
		}
		requestedBorrowers[ledger.BorrowerUesrID] = struct{}{}
	}
	rows, err := s.db.Query(
		"SELECT id, borrower_user_id FROM ledger WHERE expense_id = $1;",
		expenseID,
	)
	if err != nil {
		return err
	}

	existing := make(map[uuid.UUID]uuid.UUID)
	for rows.Next() {
		var ledgerID uuid.UUID
		var borrowerID uuid.UUID
		if err := rows.Scan(&ledgerID, &borrowerID); err != nil {
			rows.Close()
			return err
		}
		existing[borrowerID] = ledgerID
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return err
	}
	if err := rows.Close(); err != nil {
		return err
	}

	retained := make(map[uuid.UUID]struct{}, len(ledgers))
	for _, ledger := range ledgers {
		retained[ledger.BorrowerUesrID] = struct{}{}
		ledger.ExpenseID = expenseID
		ledger.LenderUserID = payerID
		if ledgerID, ok := existing[ledger.BorrowerUesrID]; ok {
			ledger.ID = ledgerID
			if err := s.UpdateLedger(ledger); err != nil {
				return err
			}
			continue
		}
		ledger.ID = uuid.New()
		if err := s.CreateLedger(ledger); err != nil {
			return err
		}
	}

	for borrowerID := range existing {
		if _, ok := retained[borrowerID]; ok {
			continue
		}
		if _, err := s.db.Exec(
			"DELETE FROM ledger WHERE expense_id = $1 AND borrower_user_id = $2;",
			expenseID,
			borrowerID,
		); err != nil {
			return err
		}
	}
	return nil
}
