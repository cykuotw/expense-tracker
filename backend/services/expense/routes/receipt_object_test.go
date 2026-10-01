package expense

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"expense-tracker/backend/services/ocr"
	"strings"
	"testing"
	"time"

	"expense-tracker/backend/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type receiptS3Fake struct {
	objects     map[string]*s3.HeadObjectOutput
	copies      int
	deletes     int
	deleteError error
}

func (f *receiptS3Fake) HeadObject(_ context.Context, input *s3.HeadObjectInput, _ ...func(*s3.Options)) (*s3.HeadObjectOutput, error) {
	object := f.objects[aws.ToString(input.Key)]
	if object == nil {
		return nil, &smithy.GenericAPIError{Code: "Forbidden"}
	}
	return object, nil
}

func (f *receiptS3Fake) CopyObject(_ context.Context, input *s3.CopyObjectInput, _ ...func(*s3.Options)) (*s3.CopyObjectOutput, error) {
	f.copies++
	destination := aws.ToString(input.Key)
	if f.objects[destination] != nil {
		return nil, &smithy.GenericAPIError{Code: "PreconditionFailed"}
	}
	for key, object := range f.objects {
		if strings.HasPrefix(key, "temporary/") {
			copy := *object
			copy.Metadata = input.Metadata
			f.objects[destination] = &copy
			return &s3.CopyObjectOutput{}, nil
		}
	}
	return nil, &smithy.GenericAPIError{Code: "NoSuchKey"}
}

func (f *receiptS3Fake) DeleteObject(_ context.Context, input *s3.DeleteObjectInput, _ ...func(*s3.Options)) (*s3.DeleteObjectOutput, error) {
	f.deletes++
	if f.deleteError != nil {
		return nil, f.deleteError
	}
	delete(f.objects, aws.ToString(input.Key))
	return &s3.DeleteObjectOutput{}, nil
}

func testReceiptObject() (types.ExpenseReceipt, *s3.HeadObjectOutput) {
	digest := strings.Repeat("a", 64)
	bytes, _ := hex.DecodeString(digest)
	receipt := types.ExpenseReceipt{
		ID: uuid.New(), ExpenseID: uuid.New(), TemporaryKey: "temporary/" + uuid.NewString() + ".jpg",
		RetainedKey: "retained/" + uuid.NewString() + "/" + uuid.NewString() + ".jpg",
		Status:      "pending", ContentType: "image/jpeg", ByteSize: 24, ChecksumSHA256: digest,
		Width: 4, Height: 6,
	}
	object := &s3.HeadObjectOutput{
		ContentLength: aws.Int64(24), ContentType: aws.String("image/jpeg"),
		ChecksumSHA256: aws.String(base64.StdEncoding.EncodeToString(bytes)),
		Metadata: map[string]string{"checksum-sha256": digest, "byte-size": "24", "width": "4", "height": "6", "content-type": "image/jpeg",
			"created-at":      receipt.SourceExpiresAt.Add(-ocr.TemporaryReceiptTTL).Format(time.RFC3339),
			"expires-at":      receipt.SourceExpiresAt.Format(time.RFC3339),
			"lifecycle-state": "temporary"},
	}
	return receipt, object
}

func TestReceiptFinalizationResumesAfterCopyAndTemporaryDeletion(t *testing.T) {
	receipt, object := testReceiptObject()
	fake := &receiptS3Fake{objects: map[string]*s3.HeadObjectOutput{receipt.TemporaryKey: object}}
	store := &receiptObjectStore{client: fake, bucket: "private-receipts"}
	require.NoError(t, store.finalize(t.Context(), receipt))
	require.Nil(t, fake.objects[receipt.TemporaryKey])
	require.NotNil(t, fake.objects[receipt.RetainedKey])
	// Simulate a database commit failure: the row is still pending, and retry
	// can finish using the verified retained object after source deletion.
	require.NoError(t, store.finalize(t.Context(), receipt))
	require.Equal(t, 1, fake.copies)
}

func TestReceiptFinalizationKeepsUnknownDestinationRetryable(t *testing.T) {
	receipt, _ := testReceiptObject()
	fake := &receiptS3Fake{objects: map[string]*s3.HeadObjectOutput{}}
	store := &receiptObjectStore{client: fake, bucket: "private-receipts"}
	require.ErrorIs(t, store.finalize(t.Context(), receipt), errReceiptObjectUnknown)
}

func TestReceiptFinalizationRejectsMismatchedDestinationWithoutOverwrite(t *testing.T) {
	receipt, object := testReceiptObject()
	wrong := *object
	wrong.ContentLength = aws.Int64(23)
	fake := &receiptS3Fake{objects: map[string]*s3.HeadObjectOutput{receipt.TemporaryKey: object, receipt.RetainedKey: &wrong}}
	store := &receiptObjectStore{client: fake, bucket: "private-receipts"}
	require.ErrorIs(t, store.finalize(t.Context(), receipt), errReceiptObjectMismatch)
	require.Zero(t, fake.copies)
}

func TestReceiptDeletionRetryIsIdempotent(t *testing.T) {
	receipt, object := testReceiptObject()
	fake := &receiptS3Fake{objects: map[string]*s3.HeadObjectOutput{receipt.RetainedKey: object}, deleteError: errors.New("transient")}
	store := &receiptObjectStore{client: fake, bucket: "private-receipts"}
	require.Error(t, store.deleteRetained(t.Context(), receipt))
	fake.deleteError = nil
	require.NoError(t, store.deleteRetained(t.Context(), receipt))
	require.NoError(t, store.deleteRetained(t.Context(), receipt))
	require.Empty(t, fake.objects)
}
