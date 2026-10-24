package ocrimprovement

import (
	"context"
	"errors"
	"strings"
	"testing"

	"expense-tracker/backend/internal/ocrtest"
	"expense-tracker/backend/services/ocr"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type cleanupObjects struct {
	keys       map[string]bool
	failDelete bool
}

func (objects *cleanupObjects) DeleteObject(_ context.Context, input *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	if objects.failDelete {
		return nil, errors.New("delete unavailable")
	}
	delete(objects.keys, *input.Key)
	return &s3.DeleteObjectOutput{}, nil
}

func (objects *cleanupObjects) ListObjectsV2(_ context.Context, input *s3.ListObjectsV2Input, _ ...func(*s3.Options)) (*s3.ListObjectsV2Output, error) {
	result := &s3.ListObjectsV2Output{}
	for key := range objects.keys {
		if strings.HasPrefix(key, *input.Prefix) {
			if len(result.Contents) >= int(aws.ToInt32(input.MaxKeys)) {
				result.IsTruncated = aws.Bool(true)
				break
			}
			result.Contents = append(result.Contents, s3types.Object{Key: aws.String(key)})
		}
	}
	return result, nil
}

func TestHistoricalAccountPurgeRetriesWithoutDeletingExpenseOrReceipt(t *testing.T) {
	fixture := ocrtest.New(t)
	run := uuid.NewString()
	_, err := fixture.DB.Exec(`INSERT INTO ocr_improvement_consent(account_id,enabled,policy_version,capture_bucket) VALUES($1,TRUE,$2,'historical-evidence')`, fixture.User, ocr.ImprovementPolicyVersion)
	require.NoError(t, err)
	_, err = fixture.DB.Exec(`INSERT INTO ocr_improvement_evidence(id,account_id,run_id,expense_id,consent_revision,policy_version,state) VALUES($1,$2,$3,$4,1,$5,'confirmed')`, uuid.New(), fixture.User, run, fixture.Expense, ocr.ImprovementPolicyVersion)
	require.NoError(t, err)
	objects := &cleanupObjects{keys: map[string]bool{
		ocr.EvidenceTemporaryKey(fixture.User.String(), run):                  true,
		ocr.EvidencePrefix(fixture.User.String(), run) + "extraction.json.gz": true,
		ocr.EvidencePrefix(fixture.User.String(), run) + "confirmed.json.gz":  true,
		"retained/product-receipt.jpg":                                        true,
	}}
	service := &Service{DB: fixture.DB, Bucket: "historical-evidence", Objects: objects}
	require.NoError(t, service.RequestAccountPurge(t.Context(), fixture.User.String()))
	require.Error(t, service.PurgeAccount(t.Context(), fixture.User.String()), "in-flight capabilities must expire first")
	_, err = fixture.DB.Exec(`UPDATE ocr_improvement_consent SET changed_at=NOW()-INTERVAL '3 minutes' WHERE account_id=$1`, fixture.User)
	require.NoError(t, err)
	objects.failDelete = true
	require.Error(t, service.PurgeAccount(t.Context(), fixture.User.String()))
	objects.failDelete = false
	require.NoError(t, service.PurgeAccount(t.Context(), fixture.User.String()))
	require.Equal(t, map[string]bool{"retained/product-receipt.jpg": true}, objects.keys)
	var enabled, pending, deleted bool
	require.NoError(t, fixture.DB.QueryRow(`SELECT enabled,purge_pending FROM ocr_improvement_consent WHERE account_id=$1`, fixture.User).Scan(&enabled, &pending))
	require.False(t, enabled)
	require.False(t, pending)
	require.NoError(t, fixture.DB.QueryRow(`SELECT is_deleted FROM expense WHERE id=$1`, fixture.Expense).Scan(&deleted))
	require.False(t, deleted)
	var count int
	require.NoError(t, fixture.DB.QueryRow(`SELECT COUNT(*) FROM ocr_improvement_evidence WHERE account_id=$1`, fixture.User).Scan(&count))
	require.Zero(t, count)
}

func TestHistoricalPurgeWithoutSamplesNeedsNoObjectStorage(t *testing.T) {
	fixture := ocrtest.New(t)
	service := &Service{DB: fixture.DB}
	require.NoError(t, service.RequestAccountPurge(t.Context(), fixture.User.String()))
	require.NoError(t, service.PurgeAccount(t.Context(), fixture.User.String()))
}
