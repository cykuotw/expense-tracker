package store

import (
	"expense-tracker/backend/types"

	"github.com/google/uuid"
)

const receiptColumns = `id, expense_id, account_id, object_id, temporary_key, retained_key,
    request_key, token_sha256, delete_request_key, role, status, content_type, byte_size, checksum_sha256, width, height,
    source_expires_at, attempt_count, COALESCE(last_error_code, ''), created_at,
    updated_at, finalized_at, deleted_at`

func (s *Store) CreateExpenseReceipt(receipt types.ExpenseReceipt) error {
	_, err := s.db.Exec(`INSERT INTO expense_receipt (
        id, expense_id, account_id, object_id, temporary_key, retained_key,
        request_key, token_sha256, role, status, content_type, byte_size, checksum_sha256,
        width, height, source_expires_at
    ) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16)`,
		receipt.ID, receipt.ExpenseID, receipt.AccountID, receipt.ObjectID,
		receipt.TemporaryKey, receipt.RetainedKey, receipt.RequestKey, receipt.TokenSHA256, receipt.Role,
		receipt.Status, receipt.ContentType, receipt.ByteSize, receipt.ChecksumSHA256,
		receipt.Width, receipt.Height, receipt.SourceExpiresAt)
	return err
}

func (s *Store) GetExpenseReceiptByRole(expenseID uuid.UUID, role string) (*types.ExpenseReceipt, error) {
	return s.getExpenseReceipt(`SELECT `+receiptColumns+` FROM expense_receipt WHERE expense_id = $1 AND role = $2`, expenseID, role)
}

func (s *Store) GetExpenseReceiptByRoleForUpdate(expenseID uuid.UUID, role string) (*types.ExpenseReceipt, error) {
	return s.getExpenseReceipt(`SELECT `+receiptColumns+` FROM expense_receipt WHERE expense_id = $1 AND role = $2 FOR UPDATE`, expenseID, role)
}

func (s *Store) GetExpenseReceiptByRequest(expenseID, requestKey uuid.UUID) (*types.ExpenseReceipt, error) {
	return s.getExpenseReceipt(`SELECT `+receiptColumns+` FROM expense_receipt WHERE expense_id = $1 AND request_key = $2`, expenseID, requestKey)
}

func (s *Store) GetExpenseReceiptForUpdate(receiptID uuid.UUID) (*types.ExpenseReceipt, error) {
	return s.getExpenseReceipt(`SELECT `+receiptColumns+` FROM expense_receipt WHERE id = $1 FOR UPDATE`, receiptID)
}

func (s *Store) getExpenseReceipt(query string, args ...any) (*types.ExpenseReceipt, error) {
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	if !rows.Next() {
		return nil, rows.Err()
	}
	receipt := new(types.ExpenseReceipt)
	err = rows.Scan(&receipt.ID, &receipt.ExpenseID, &receipt.AccountID, &receipt.ObjectID,
		&receipt.TemporaryKey, &receipt.RetainedKey, &receipt.RequestKey, &receipt.TokenSHA256, &receipt.DeleteRequestKey, &receipt.Role,
		&receipt.Status, &receipt.ContentType, &receipt.ByteSize, &receipt.ChecksumSHA256,
		&receipt.Width, &receipt.Height, &receipt.SourceExpiresAt, &receipt.AttemptCount,
		&receipt.LastErrorCode, &receipt.CreatedAt, &receipt.UpdatedAt,
		&receipt.FinalizedAt, &receipt.DeletedAt)
	if err != nil {
		return nil, err
	}
	return receipt, rows.Err()
}

func (s *Store) UpdateExpenseReceipt(receipt types.ExpenseReceipt) error {
	_, err := s.db.Exec(`UPDATE expense_receipt SET role = $2, status = $3,
        attempt_count = $4, last_error_code = NULLIF($5, ''), updated_at = NOW(),
        finalized_at = $6, deleted_at = $7, delete_request_key = $8 WHERE id = $1`, receipt.ID, receipt.Role,
		receipt.Status, receipt.AttemptCount, receipt.LastErrorCode,
		receipt.FinalizedAt, receipt.DeletedAt, receipt.DeleteRequestKey)
	return err
}

func (s *Store) GetExpenseReceiptByDeleteRequest(expenseID, requestKey uuid.UUID) (*types.ExpenseReceipt, error) {
	return s.getExpenseReceipt(`SELECT `+receiptColumns+` FROM expense_receipt WHERE expense_id = $1 AND delete_request_key = $2`, expenseID, requestKey)
}

func (s *Store) ListExpenseReceiptsForCleanup(expenseID uuid.UUID) ([]types.ExpenseReceipt, error) {
	rows, err := s.db.Query(`SELECT `+receiptColumns+` FROM expense_receipt WHERE expense_id = $1 AND status = 'deleting' ORDER BY id`, expenseID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []types.ExpenseReceipt{}
	for rows.Next() {
		var receipt types.ExpenseReceipt
		if err := rows.Scan(&receipt.ID, &receipt.ExpenseID, &receipt.AccountID, &receipt.ObjectID,
			&receipt.TemporaryKey, &receipt.RetainedKey, &receipt.RequestKey, &receipt.TokenSHA256, &receipt.DeleteRequestKey,
			&receipt.Role, &receipt.Status, &receipt.ContentType, &receipt.ByteSize,
			&receipt.ChecksumSHA256, &receipt.Width, &receipt.Height, &receipt.SourceExpiresAt,
			&receipt.AttemptCount, &receipt.LastErrorCode, &receipt.CreatedAt, &receipt.UpdatedAt,
			&receipt.FinalizedAt, &receipt.DeletedAt); err != nil {
			return nil, err
		}
		result = append(result, receipt)
	}
	return result, rows.Err()
}
