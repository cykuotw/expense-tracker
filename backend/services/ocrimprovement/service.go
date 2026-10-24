// Package ocrimprovement provides cleanup of historical evidence only.
// New samples and public improvement APIs are no longer supported.
package ocrimprovement

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"expense-tracker/backend/services/ocr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
)

var errUnavailable = errors.New("historical evidence cleanup is temporarily unavailable")

type Objects interface {
	DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	ListObjectsV2(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

type Service struct {
	DB      *sql.DB
	Objects Objects
	Bucket  string
}

// RequestAccountPurge is the operator/account-deletion hook. It revokes capture
// first; PurgeAccount then executes bounded, retryable object deletion.
func (s *Service) RequestAccountPurge(ctx context.Context, id string) error {
	if _, err := uuid.Parse(id); err != nil {
		return err
	}
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	_, err = tx.ExecContext(ctx, `INSERT INTO ocr_improvement_consent(account_id,enabled,policy_version,purge_pending) VALUES($1,FALSE,$2,TRUE) ON CONFLICT(account_id) DO UPDATE SET enabled=FALSE,revision=ocr_improvement_consent.revision+1,purge_pending=TRUE,changed_at=NOW()`, id, ocr.ImprovementPolicyVersion)
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE ocr_improvement_evidence SET state='deleting',updated_at=NOW() WHERE account_id=$1 AND state<>'deleted'`, id)
	if err != nil {
		return err
	}
	return tx.Commit()
}

// PurgeAccount never deletes product receipts or expense values. Callers retry
// until pending clears after already-issued capabilities have expired.
func (s *Service) PurgeAccount(ctx context.Context, id string) error {
	work, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	tx, err := s.DB.BeginTx(work, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var pending bool
	var changed time.Time
	var captureBucket string
	err = tx.QueryRowContext(work, `SELECT purge_pending,changed_at,capture_bucket FROM ocr_improvement_consent WHERE account_id=$1 FOR UPDATE`, id).Scan(&pending, &changed, &captureBucket)
	if errors.Is(err, sql.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if !pending {
		return nil
	}
	if captureBucket == "" {
		var evidenceExists bool
		if err := tx.QueryRowContext(work, `SELECT EXISTS(SELECT 1 FROM ocr_improvement_evidence WHERE account_id=$1 AND state<>'deleted')`, id).Scan(&evidenceExists); err != nil {
			return err
		}
		if evidenceExists {
			return errUnavailable
		}
		if _, err := tx.ExecContext(work, `DELETE FROM ocr_improvement_evidence WHERE account_id=$1`, id); err != nil {
			return err
		}
		// No capture-enabled capability was issued and no bundle exists.
		if _, err := tx.ExecContext(work, `UPDATE ocr_improvement_consent SET purge_pending=FALSE WHERE account_id=$1`, id); err != nil {
			return err
		}
		return tx.Commit()
	}
	if s.Objects == nil || s.Bucket == "" || s.Bucket != captureBucket {
		return errUnavailable
	}
	// Let already-issued 60-second capabilities finish before verifying deletion.
	if time.Since(changed) < ocr.CapabilityTTL+45*time.Second {
		return errUnavailable
	}
	for _, prefix := range []string{"temporary/" + id + "/", "confirmed/" + id + "/"} {
		listing, err := s.Objects.ListObjectsV2(work, &s3.ListObjectsV2Input{Bucket: aws.String(s.Bucket), Prefix: aws.String(prefix), MaxKeys: aws.Int32(100)})
		if err != nil {
			return err
		}
		for _, object := range listing.Contents {
			if _, err := s.Objects.DeleteObject(work, &s3.DeleteObjectInput{Bucket: aws.String(s.Bucket), Key: object.Key}); err != nil {
				return err
			}
		}
		if aws.ToBool(listing.IsTruncated) {
			return errUnavailable
		}
		check, err := s.Objects.ListObjectsV2(work, &s3.ListObjectsV2Input{Bucket: aws.String(s.Bucket), Prefix: aws.String(prefix), MaxKeys: aws.Int32(1)})
		if err != nil || len(check.Contents) > 0 {
			return errUnavailable
		}
	}
	if _, err = tx.ExecContext(work, `DELETE FROM ocr_improvement_evidence WHERE account_id=$1`, id); err != nil {
		return err
	}
	if _, err = tx.ExecContext(work, `UPDATE ocr_improvement_consent SET purge_pending=FALSE,capture_bucket='' WHERE account_id=$1`, id); err != nil {
		return err
	}
	return tx.Commit()
}

// PurgeExpense can be called after receipt removal or permanent expense purge.
func (s *Service) PurgeExpense(ctx context.Context, expenseID string) error {
	if _, err := s.DB.ExecContext(ctx, `UPDATE ocr_improvement_evidence SET state='deleting',updated_at=NOW() WHERE expense_id=$1 AND state<>'deleted'`, expenseID); err != nil {
		return err
	}
	return s.CleanupExpense(ctx, expenseID)
}

// CleanupExpense resumes marked evidence deletion without requesting removal of
// a still-current receipt or capturing any new provider output.
func (s *Service) CleanupExpense(ctx context.Context, expenseID string) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	rows, err := s.DB.QueryContext(ctx, `SELECT id,account_id,run_id FROM ocr_improvement_evidence WHERE expense_id=$1 AND state='deleting' ORDER BY id LIMIT 20`, expenseID)
	if err != nil {
		return err
	}
	type bundle struct{ id, account, run string }
	bundles := []bundle{}
	for rows.Next() {
		var b bundle
		if err := rows.Scan(&b.id, &b.account, &b.run); err != nil {
			rows.Close()
			return err
		}
		bundles = append(bundles, b)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if len(bundles) == 0 {
		return nil
	}
	if s.Objects == nil {
		return errUnavailable
	}
	for _, b := range bundles {
		for _, key := range []string{ocr.EvidenceTemporaryKey(b.account, b.run), ocr.EvidencePrefix(b.account, b.run) + "extraction.json.gz", ocr.EvidencePrefix(b.account, b.run) + "confirmed.json.gz"} {
			if _, err := s.Objects.DeleteObject(ctx, &s3.DeleteObjectInput{Bucket: aws.String(s.Bucket), Key: aws.String(key)}); err != nil {
				return err
			}
		}
		for _, prefix := range []string{ocr.EvidenceTemporaryKey(b.account, b.run), ocr.EvidencePrefix(b.account, b.run)} {
			listing, err := s.Objects.ListObjectsV2(ctx, &s3.ListObjectsV2Input{Bucket: aws.String(s.Bucket), Prefix: aws.String(prefix), MaxKeys: aws.Int32(1)})
			if err != nil || len(listing.Contents) > 0 {
				return errUnavailable
			}
		}
		if _, err := s.DB.ExecContext(ctx, `UPDATE ocr_improvement_evidence SET state='deleted',expense_id=NULL,receipt_id=NULL,metadata='{}'::jsonb,updated_at=NOW() WHERE id=$1`, b.id); err != nil {
			return err
		}
	}
	var remaining bool
	if err := s.DB.QueryRowContext(ctx, `SELECT EXISTS(SELECT 1 FROM ocr_improvement_evidence WHERE expense_id=$1 AND state='deleting')`, expenseID).Scan(&remaining); err != nil {
		return err
	}
	if remaining {
		return errUnavailable
	}
	return nil
}
