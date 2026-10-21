package user

import (
	"context"
	"database/sql"
	"errors"
	"expense-tracker/backend/services/ocr"
	"os"
)

func (s *Store) GetOCRConsentRevision(ctx context.Context, accountID string) (int64, error) {
	bucket := os.Getenv("OCR_EVIDENCE_BUCKET")
	if os.Getenv("OCR_EVIDENCE_CAPTURE_ENABLED") != "true" || bucket == "" {
		return 0, nil
	}
	var revision int64
	// Record that a capability can stage data before signing it. Opt-out locks
	// this row too, so purge cannot incorrectly declare an unused bucket empty.
	err := s.db.QueryRowContext(ctx, `UPDATE ocr_improvement_consent SET capture_bucket=$3 WHERE account_id=$1 AND enabled AND policy_version=$2 AND NOT purge_pending AND capture_bucket IN ('',$3) RETURNING revision`, accountID, ocr.ImprovementPolicyVersion, bucket).Scan(&revision)
	if errors.Is(err, sql.ErrNoRows) {
		return 0, nil
	}
	return revision, err
}
