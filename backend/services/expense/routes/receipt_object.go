package expense

import (
	"context"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"expense-tracker/backend/services/ocr"
	"expense-tracker/backend/types"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	s3types "github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
)

var (
	errReceiptObjectMissing  = errors.New("receipt object missing")
	errReceiptObjectUnknown  = errors.New("receipt object availability unknown")
	errReceiptObjectMismatch = errors.New("receipt object metadata mismatch")
)

type receiptS3API interface {
	HeadObject(context.Context, *s3.HeadObjectInput, ...func(*s3.Options)) (*s3.HeadObjectOutput, error)
	CopyObject(context.Context, *s3.CopyObjectInput, ...func(*s3.Options)) (*s3.CopyObjectOutput, error)
	DeleteObject(context.Context, *s3.DeleteObjectInput, ...func(*s3.Options)) (*s3.DeleteObjectOutput, error)
}

type receiptObjectStore struct {
	client receiptS3API
	bucket string
}

func (s *receiptObjectStore) head(ctx context.Context, key string) (*s3.HeadObjectOutput, error) {
	output, err := s.client.HeadObject(ctx, &s3.HeadObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(key), ChecksumMode: s3types.ChecksumModeEnabled,
	})
	if err == nil {
		return output, nil
	}
	var apiErr smithy.APIError
	if errors.As(err, &apiErr) {
		switch apiErr.ErrorCode() {
		case "NotFound", "NoSuchKey", "404":
			return nil, errReceiptObjectMissing
		case "Forbidden", "AccessDenied", "403":
			return nil, errReceiptObjectUnknown
		}
	}
	return nil, err
}

func (s *receiptObjectStore) verify(ctx context.Context, key string, receipt types.ExpenseReceipt) error {
	object, err := s.head(ctx, key)
	if err != nil {
		return err
	}
	digest, err := hex.DecodeString(receipt.ChecksumSHA256)
	if err != nil {
		return errReceiptObjectMismatch
	}
	expectedChecksum := base64.StdEncoding.EncodeToString(digest)
	expectedState := "temporary"
	if strings.HasPrefix(key, "retained/") {
		expectedState = "retained"
	}
	if object.Metadata["lifecycle-state"] != expectedState ||
		object.Metadata["created-at"] != receipt.SourceExpiresAt.Add(-ocr.TemporaryReceiptTTL).UTC().Format(time.RFC3339) {
		return errReceiptObjectMismatch
	}
	if expectedState == "temporary" && object.Metadata["expires-at"] != receipt.SourceExpiresAt.UTC().Format(time.RFC3339) {
		return errReceiptObjectMismatch
	}
	if expectedState == "retained" && object.Metadata["source-expires-at"] != receipt.SourceExpiresAt.UTC().Format(time.RFC3339) {
		return errReceiptObjectMismatch
	}
	if aws.ToInt64(object.ContentLength) != receipt.ByteSize ||
		aws.ToString(object.ContentType) != receipt.ContentType ||
		aws.ToString(object.ChecksumSHA256) != expectedChecksum ||
		object.Metadata["checksum-sha256"] != receipt.ChecksumSHA256 ||
		object.Metadata["byte-size"] != strconv.FormatInt(receipt.ByteSize, 10) ||
		object.Metadata["width"] != strconv.Itoa(receipt.Width) ||
		object.Metadata["height"] != strconv.Itoa(receipt.Height) ||
		object.Metadata["content-type"] != receipt.ContentType {
		return errReceiptObjectMismatch
	}
	return nil
}

func (s *receiptObjectStore) verifyTemporary(ctx context.Context, receipt types.ExpenseReceipt) error {
	return s.verify(ctx, receipt.TemporaryKey, receipt)
}

func (s *receiptObjectStore) finalize(ctx context.Context, receipt types.ExpenseReceipt) error {
	destinationErr := s.verify(ctx, receipt.RetainedKey, receipt)
	if destinationErr != nil {
		if !errors.Is(destinationErr, errReceiptObjectMissing) && !errors.Is(destinationErr, errReceiptObjectUnknown) {
			return destinationErr
		}
		sourceErr := s.verifyTemporary(ctx, receipt)
		if sourceErr != nil && !errors.Is(sourceErr, errReceiptObjectUnknown) {
			if errors.Is(destinationErr, errReceiptObjectUnknown) {
				return errReceiptObjectUnknown
			}
			return sourceErr
		}
		_, err := s.client.CopyObject(ctx, &s3.CopyObjectInput{
			Bucket: aws.String(s.bucket), Key: aws.String(receipt.RetainedKey),
			CopySource:           aws.String(url.PathEscape(s.bucket + "/" + receipt.TemporaryKey)),
			IfNoneMatch:          aws.String("*"),
			MetadataDirective:    s3types.MetadataDirectiveReplace,
			ChecksumAlgorithm:    s3types.ChecksumAlgorithmSha256,
			ServerSideEncryption: s3types.ServerSideEncryptionAes256,
			ContentType:          aws.String(receipt.ContentType),
			CacheControl:         aws.String("private, no-store"),
			Metadata: map[string]string{
				"content-type":      receipt.ContentType,
				"byte-size":         strconv.FormatInt(receipt.ByteSize, 10),
				"checksum-sha256":   receipt.ChecksumSHA256,
				"width":             strconv.Itoa(receipt.Width),
				"height":            strconv.Itoa(receipt.Height),
				"created-at":        receipt.SourceExpiresAt.Add(-ocr.TemporaryReceiptTTL).UTC().Format(time.RFC3339),
				"source-expires-at": receipt.SourceExpiresAt.UTC().Format(time.RFC3339),
				"lifecycle-state":   "retained",
			},
		})
		if err != nil {
			var apiErr smithy.APIError
			if errors.As(err, &apiErr) {
				if apiErr.ErrorCode() == "NoSuchKey" || apiErr.ErrorCode() == "NotFound" {
					if errors.Is(destinationErr, errReceiptObjectUnknown) {
						return errReceiptObjectUnknown
					}
					return errReceiptObjectMissing
				}
				if strings.EqualFold(apiErr.ErrorCode(), "PreconditionFailed") {
					// Another attempt already placed the deterministic destination.
				} else {
					return fmt.Errorf("copy receipt: %w", err)
				}
			} else {
				return fmt.Errorf("copy receipt: %w", err)
			}
		}
		if err := s.verify(ctx, receipt.RetainedKey, receipt); err != nil {
			return err
		}
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(receipt.TemporaryKey),
	})
	return err
}

func (s *receiptObjectStore) deleteRetained(ctx context.Context, receipt types.ExpenseReceipt) error {
	if _, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(receipt.RetainedKey),
	}); err != nil {
		return err
	}
	_, err := s.client.DeleteObject(ctx, &s3.DeleteObjectInput{
		Bucket: aws.String(s.bucket), Key: aws.String(receipt.TemporaryKey),
	})
	return err
}

func receiptFromToken(expenseID, requestKey uuid.UUID, temporary ocr.TemporaryReceipt) (types.ExpenseReceipt, error) {
	accountID, err := uuid.Parse(temporary.AccountID)
	if err != nil {
		return types.ExpenseReceipt{}, err
	}
	objectID, err := uuid.Parse(temporary.ObjectID)
	if err != nil {
		return types.ExpenseReceipt{}, err
	}
	return types.ExpenseReceipt{
		ID: uuid.New(), ExpenseID: expenseID, AccountID: accountID, ObjectID: objectID,
		TemporaryKey: temporary.Key,
		RetainedKey:  "retained/" + expenseID.String() + "/" + objectID.String() + ".jpg",
		RequestKey:   requestKey, Role: "current", Status: "pending",
		ContentType: temporary.ContentType, ByteSize: temporary.ByteSize,
		ChecksumSHA256: temporary.ChecksumSHA256, Width: temporary.Width, Height: temporary.Height,
		SourceExpiresAt: temporary.ExpiresAt,
	}, nil
}
