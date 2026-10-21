package ocrimprovement

import (
	"bytes"
	"compress/gzip"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/services/ocr"
	"expense-tracker/backend/utils"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var errUnavailable = errors.New("improvement evidence is temporarily unavailable")
var errConsent = errors.New("improvement consent is absent or outdated")

func parseRequest(c *gin.Context, payload any) error {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 8192)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(payload); err != nil {
		return err
	}
	if err := decoder.Decode(new(any)); !errors.Is(err, io.EOF) {
		return errConsent
	}
	return nil
}

type Objects interface {
	HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
	DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
	ListObjectsV2(context.Context, *s3.ListObjectsV2Input, ...func(*s3.Options)) (*s3.ListObjectsV2Output, error)
}

type Service struct {
	DB             *sql.DB
	Objects        Objects
	Bucket         string
	Secret         []byte
	CaptureEnabled bool
}
type Preference struct {
	Enabled          bool       `json:"enabled"`
	PolicyVersion    string     `json:"policyVersion"`
	Revision         int64      `json:"revision"`
	PurgePending     bool       `json:"purgePending"`
	ChangedAt        *time.Time `json:"changedAt,omitempty"`
	CaptureAvailable bool       `json:"captureAvailable"`
}

func (s *Service) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/account/ocr-improvement", s.preference)
	router.PATCH("/account/ocr-improvement", s.update)
	router.POST("/account/ocr-improvement/purge", s.purge)
	router.POST("/expense/:expenseId/ocr-evidence", s.confirm)
}

func (s *Service) account(c *gin.Context) (string, bool) {
	c.Header("Cache-Control", "private, no-store")
	id, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, errors.New("authentication required"))
		return "", false
	}
	var active bool
	if err := s.DB.QueryRowContext(c.Request.Context(), `SELECT is_active FROM users WHERE id=$1`, id).Scan(&active); err != nil || !active {
		utils.WriteError(c, 403, errConsent)
		return "", false
	}
	return id, true
}

func (s *Service) Preference(ctx context.Context, id string) (Preference, error) {
	p := Preference{PolicyVersion: ocr.ImprovementPolicyVersion, CaptureAvailable: s.CaptureEnabled}
	err := s.DB.QueryRowContext(ctx, `SELECT enabled,policy_version,revision,purge_pending,changed_at FROM ocr_improvement_consent WHERE account_id=$1`, id).Scan(&p.Enabled, &p.PolicyVersion, &p.Revision, &p.PurgePending, &p.ChangedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return p, nil
	}
	if p.PolicyVersion != ocr.ImprovementPolicyVersion {
		p.Enabled = false
	}
	return p, err
}

func (s *Service) preference(c *gin.Context) {
	id, ok := s.account(c)
	if !ok {
		return
	}
	p, err := s.Preference(c.Request.Context(), id)
	if err != nil {
		utils.WriteError(c, 503, errUnavailable)
		return
	}
	utils.WriteJSON(c, 200, p)
}

func (s *Service) update(c *gin.Context) {
	id, ok := s.account(c)
	if !ok {
		return
	}
	var payload struct {
		Enabled          *bool  `json:"enabled"`
		PolicyVersion    string `json:"policyVersion"`
		ExpectedRevision int64  `json:"expectedRevision"`
	}
	if parseRequest(c, &payload) != nil || payload.Enabled == nil || payload.PolicyVersion != ocr.ImprovementPolicyVersion {
		utils.WriteError(c, 400, errConsent)
		return
	}
	tx, err := s.DB.BeginTx(c.Request.Context(), nil)
	if err != nil {
		utils.WriteError(c, 503, errUnavailable)
		return
	}
	defer func() { _ = tx.Rollback() }()
	var active, granted bool
	err = tx.QueryRowContext(c.Request.Context(), `SELECT is_active,EXISTS(SELECT 1 FROM user_feature_grant WHERE user_id=users.id AND feature_key='receipt_ocr') FROM users WHERE id=$1 FOR UPDATE`, id).Scan(&active, &granted)
	if err != nil || !active || (*payload.Enabled && !granted) {
		utils.WriteError(c, 403, errConsent)
		return
	}
	_, err = tx.ExecContext(c.Request.Context(), `INSERT INTO ocr_improvement_consent(account_id,enabled,policy_version) VALUES($1,FALSE,$2) ON CONFLICT DO NOTHING`, id, ocr.ImprovementPolicyVersion)
	if err != nil {
		utils.WriteError(c, 503, errUnavailable)
		return
	}
	var revision int64
	var pending bool
	err = tx.QueryRowContext(c.Request.Context(), `SELECT revision,purge_pending FROM ocr_improvement_consent WHERE account_id=$1 FOR UPDATE`, id).Scan(&revision, &pending)
	// The first stored revision is 1, while an absent preference is presented as 0.
	if err != nil || (payload.ExpectedRevision != revision && !(payload.ExpectedRevision == 0 && revision == 1)) || (*payload.Enabled && pending) {
		utils.WriteError(c, 409, errConsent)
		return
	}
	_, err = tx.ExecContext(c.Request.Context(), `UPDATE ocr_improvement_consent SET enabled=$2,policy_version=$3,revision=revision+1,purge_pending=purge_pending OR NOT $2,changed_at=NOW() WHERE account_id=$1`, id, *payload.Enabled, ocr.ImprovementPolicyVersion)
	if err == nil && !*payload.Enabled {
		_, err = tx.ExecContext(c.Request.Context(), `UPDATE ocr_improvement_evidence SET state='deleting',updated_at=NOW() WHERE account_id=$1 AND state<>'deleted'`, id)
	}
	if err != nil || tx.Commit() != nil {
		utils.WriteError(c, 503, errUnavailable)
		return
	}
	if !*payload.Enabled {
		_ = s.PurgeAccount(c.Request.Context(), id)
	}
	p, err := s.Preference(c.Request.Context(), id)
	if err != nil {
		utils.WriteError(c, 503, errUnavailable)
		return
	}
	status := 200
	if p.PurgePending {
		status = 202
	}
	utils.WriteJSON(c, status, p)
}

func (s *Service) purge(c *gin.Context) {
	id, ok := s.account(c)
	if !ok {
		return
	}
	err := s.PurgeAccount(c.Request.Context(), id)
	p, readErr := s.Preference(c.Request.Context(), id)
	if readErr != nil {
		utils.WriteError(c, 503, errUnavailable)
		return
	}
	status := 200
	if err != nil || p.PurgePending {
		status = 202
	}
	utils.WriteJSON(c, status, p)
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

func (s *Service) confirm(c *gin.Context) {
	id, ok := s.account(c)
	if !ok {
		return
	}
	expense, err := uuid.Parse(c.Param("expenseId"))
	if err != nil {
		utils.WriteError(c, 404, errConsent)
		return
	}
	var payload struct {
		Token string `json:"token"`
	}
	if parseRequest(c, &payload) != nil {
		utils.WriteError(c, 400, errConsent)
		return
	}
	claims, err := ocr.VerifyEvidenceToken(s.Secret, payload.Token, id, time.Now())
	if err != nil {
		utils.WriteError(c, 400, errConsent)
		return
	}
	if !s.CaptureEnabled || s.Objects == nil || s.Bucket == "" {
		utils.WriteJSON(c, 200, gin.H{"status": "not_captured"})
		return
	}
	err = s.Confirm(c.Request.Context(), id, expense, claims)
	if err != nil {
		utils.WriteError(c, 503, errUnavailable)
		return
	}
	utils.WriteJSON(c, 200, gin.H{"status": "captured"})
}

func (s *Service) Confirm(ctx context.Context, account string, expense uuid.UUID, claims *ocr.EvidenceClaims) error {
	if !s.CaptureEnabled || s.Objects == nil || s.Bucket == "" {
		return errUnavailable
	}
	if claims == nil || claims.Subject != account || claims.Revision <= 0 {
		return errConsent
	}
	ctx, cancel := context.WithTimeout(ctx, 6*time.Second)
	defer cancel()
	tx, err := s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	var enabled, pending bool
	var revision int64
	var policy string
	err = tx.QueryRowContext(ctx, `SELECT enabled,revision,policy_version,purge_pending FROM ocr_improvement_consent WHERE account_id=$1 FOR UPDATE`, account).Scan(&enabled, &revision, &policy, &pending)
	if err != nil || !enabled || pending || revision != claims.Revision || policy != ocr.ImprovementPolicyVersion {
		return errConsent
	}
	var snapshot json.RawMessage
	err = tx.QueryRowContext(ctx, `SELECT jsonb_build_object('merchant',e.provider_name,'date',COALESCE(e.occurred_on,(e.expense_time_utc AT TIME ZONE 'UTC')::date)::text,'currency',e.currency,'subtotal',e.sub_total::text,'taxFeeTip',e.tax_fee_tip::text,'total',e.total::text,'items',COALESCE((SELECT jsonb_agg(jsonb_build_object('description',COALESCE(i.description,i.name),'quantity',(CASE WHEN i.description IS NULL THEN i.amount ELSE i.quantity END)::text,'unitPrice',(CASE WHEN i.description IS NULL THEN i.unit_price ELSE i.confirmed_unit_price END)::text,'lineTotal',COALESCE(i.line_total,ROUND(i.amount*i.unit_price,3))::text) ORDER BY i.position) FROM item i WHERE i.expense_id=e.id),'[]'::jsonb)) FROM expense e JOIN users u ON u.id=$2 WHERE e.id=$1 AND NOT e.is_deleted AND u.is_active AND e.create_by_user_id=$2 AND EXISTS(SELECT 1 FROM group_member gm WHERE gm.group_id=e.group_id AND gm.user_id=$2) AND EXISTS(SELECT 1 FROM user_feature_grant WHERE user_id=$2 AND feature_key='receipt_ocr') FOR SHARE OF e`, expense, account).Scan(&snapshot)
	if err != nil {
		return errConsent
	}
	run, err := uuid.Parse(claims.ID)
	if err != nil {
		return errConsent
	}
	var existingExpense uuid.UUID
	var state string
	err = tx.QueryRowContext(ctx, `SELECT expense_id,state FROM ocr_improvement_evidence WHERE run_id=$1`, run).Scan(&existingExpense, &state)
	if err == nil {
		if existingExpense != expense || state == "deleted" || state == "deleting" {
			return errConsent
		}
		if state == "confirmed" {
			return nil
		}
	} else if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	temporary := ocr.EvidenceTemporaryKey(account, claims.ID)
	prefix := ocr.EvidencePrefix(account, claims.ID)
	head, err := s.Objects.HeadObject(ctx, &s3.HeadObjectInput{Bucket: aws.String(s.Bucket), Key: aws.String(temporary)})
	if err != nil || head.Metadata["sha256"] != claims.Checksum || head.Metadata["consent-revision"] != strconv.FormatInt(claims.Revision, 10) {
		return errConsent
	}
	metadata, _ := json.Marshal(head.Metadata)
	// Record the operation before writing persistent evidence so retries and
	// opt-out can always enumerate partially written bundles.
	_, err = tx.ExecContext(ctx, `INSERT INTO ocr_improvement_evidence(id,account_id,run_id,expense_id,consent_revision,policy_version,state,metadata) VALUES($1,$2,$3,$4,$5,$6,'pending',$7) ON CONFLICT(run_id) DO NOTHING`, uuid.New(), account, run, expense, claims.Revision, policy, metadata)
	if err != nil {
		return err
	}
	if err = tx.Commit(); err != nil {
		return err
	}
	// A second transaction rechecks consent while serializing writes with purge.
	tx, err = s.DB.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()
	err = tx.QueryRowContext(ctx, `SELECT enabled,revision,purge_pending FROM ocr_improvement_consent WHERE account_id=$1 FOR UPDATE`, account).Scan(&enabled, &revision, &pending)
	if err != nil || !enabled || pending || revision != claims.Revision {
		return errConsent
	}
	err = tx.QueryRowContext(ctx, `SELECT state FROM ocr_improvement_evidence WHERE run_id=$1 FOR UPDATE`, run).Scan(&state)
	if err != nil || state == "deleted" || state == "deleting" {
		return errConsent
	}
	if state == "confirmed" {
		return nil
	}
	_, err = s.Objects.CopyObject(ctx, &s3.CopyObjectInput{Bucket: aws.String(s.Bucket), Key: aws.String(prefix + "extraction.json.gz"), CopySource: aws.String(url.PathEscape(s.Bucket + "/" + temporary)), ServerSideEncryption: s3types.ServerSideEncryptionAes256})
	if err != nil {
		return err
	}
	var compressed bytes.Buffer
	zipper := gzip.NewWriter(&compressed)
	if _, err = zipper.Write(snapshot); err != nil {
		return err
	}
	if err = zipper.Close(); err != nil {
		return err
	}
	_, err = s.Objects.PutObject(ctx, &s3.PutObjectInput{Bucket: aws.String(s.Bucket), Key: aws.String(prefix + "confirmed.json.gz"), Body: bytes.NewReader(compressed.Bytes()), ContentType: aws.String("application/gzip"), CacheControl: aws.String("private, no-store"), ServerSideEncryption: s3types.ServerSideEncryptionAes256})
	if err != nil {
		return err
	}
	_, err = tx.ExecContext(ctx, `UPDATE ocr_improvement_evidence SET state='confirmed',updated_at=NOW() WHERE run_id=$1 AND state='pending'`, run)
	if err != nil {
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
