package store

import (
	"expense-tracker/backend/types"
	"fmt"
)

func (s *Store) ClaimExpenseCreateIdempotency(record types.ExpenseCreateIdempotency) (types.ExpenseCreateIdempotency, bool, error) {
	rows, err := s.db.Query(
		`INSERT INTO expense_create_idempotency (
			creator_user_id, idempotency_key, request_fingerprint, expense_id
		) VALUES ($1, $2, $3, $4)
		ON CONFLICT (creator_user_id, idempotency_key) DO NOTHING
		RETURNING request_fingerprint, expense_id`,
		record.CreatorUserID,
		record.Key,
		record.RequestFingerprint,
		record.ExpenseID,
	)
	if err != nil {
		return types.ExpenseCreateIdempotency{}, false, err
	}
	defer rows.Close()

	if rows.Next() {
		var claimed types.ExpenseCreateIdempotency
		if err := rows.Scan(&claimed.RequestFingerprint, &claimed.ExpenseID); err != nil {
			return types.ExpenseCreateIdempotency{}, false, err
		}
		claimed.CreatorUserID = record.CreatorUserID
		claimed.Key = record.Key
		return claimed, true, rows.Err()
	}
	if err := rows.Err(); err != nil {
		return types.ExpenseCreateIdempotency{}, false, err
	}

	rows, err = s.db.Query(
		`SELECT request_fingerprint, expense_id
		FROM expense_create_idempotency
		WHERE creator_user_id = $1 AND idempotency_key = $2`,
		record.CreatorUserID,
		record.Key,
	)
	if err != nil {
		return types.ExpenseCreateIdempotency{}, false, err
	}
	defer rows.Close()
	if !rows.Next() {
		if err := rows.Err(); err != nil {
			return types.ExpenseCreateIdempotency{}, false, err
		}
		return types.ExpenseCreateIdempotency{}, false, fmt.Errorf("idempotency record disappeared after a conflict")
	}

	var existing types.ExpenseCreateIdempotency
	if err := rows.Scan(&existing.RequestFingerprint, &existing.ExpenseID); err != nil {
		return types.ExpenseCreateIdempotency{}, false, err
	}
	existing.CreatorUserID = record.CreatorUserID
	existing.Key = record.Key
	return existing, false, rows.Err()
}
