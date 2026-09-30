package ocr

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
)

var ErrInvalidTemporaryReceipt = errors.New("invalid temporary receipt")

type ReceiptPutObjectAPI interface {
	PutObject(context.Context, *s3.PutObjectInput, ...func(*s3.Options)) (*s3.PutObjectOutput, error)
}

// TemporaryReceiptStore has only a PutObject dependency; it cannot read images.
type TemporaryReceiptStore struct {
	client ReceiptPutObjectAPI
	bucket string
	secret []byte
}

func NewTemporaryReceiptStore(client ReceiptPutObjectAPI, bucket string, secret []byte) (*TemporaryReceiptStore, error) {
	if client == nil || bucket == "" || len(secret) < 32 {
		return nil, ErrInvalidTemporaryReceipt
	}
	return &TemporaryReceiptStore{client: client, bucket: bucket, secret: bytes.Clone(secret)}, nil
}

// Store accepts only a successful server-side preprocessing result.
func (store *TemporaryReceiptStore) Store(ctx context.Context, accountID string, image PreprocessedImage, now time.Time) (TemporaryReceipt, string, error) {
	if !image.validated || image.Width <= 0 || image.Height <= 0 ||
		len(image.Bytes) == 0 || len(image.Bytes) > DefaultMaxOutputBytes {
		return TemporaryReceipt{}, "", ErrInvalidTemporaryReceipt
	}
	objectID, err := uuid.NewRandom()
	if err != nil {
		return TemporaryReceipt{}, "", fmt.Errorf("generate receipt identity: %w", err)
	}
	key, err := TemporaryReceiptKey(objectID.String())
	if err != nil {
		return TemporaryReceipt{}, "", err
	}
	digest := sha256.Sum256(image.Bytes)
	created := now.UTC().Truncate(time.Second)
	receipt := TemporaryReceipt{
		AccountID: accountID, ObjectID: objectID.String(), Key: key,
		ChecksumSHA256: hex.EncodeToString(digest[:]), ContentType: "image/jpeg",
		ByteSize: int64(len(image.Bytes)), Width: image.Width, Height: image.Height,
		CreatedAt: created, ExpiresAt: created.Add(TemporaryReceiptTTL),
	}
	token, err := MintReceiptToken(store.secret, receipt)
	if err != nil {
		return TemporaryReceipt{}, "", err
	}
	_, err = store.client.PutObject(ctx, &s3.PutObjectInput{
		Bucket: aws.String(store.bucket), Key: aws.String(receipt.Key),
		Body: bytes.NewReader(image.Bytes), ContentLength: aws.Int64(receipt.ByteSize),
		ContentType:          aws.String(receipt.ContentType),
		ChecksumSHA256:       aws.String(base64.StdEncoding.EncodeToString(digest[:])),
		ServerSideEncryption: types.ServerSideEncryptionAes256,
		IfNoneMatch:          aws.String("*"), CacheControl: aws.String("private, no-store"),
		Metadata: map[string]string{
			"content-type":    receipt.ContentType,
			"byte-size":       strconv.FormatInt(receipt.ByteSize, 10),
			"checksum-sha256": receipt.ChecksumSHA256,
			"width":           strconv.Itoa(receipt.Width), "height": strconv.Itoa(receipt.Height),
			"created-at":      receipt.CreatedAt.Format(time.RFC3339),
			"expires-at":      receipt.ExpiresAt.Format(time.RFC3339),
			"lifecycle-state": "temporary",
		},
	})
	if err != nil {
		return TemporaryReceipt{}, "", fmt.Errorf("write temporary receipt: %w", err)
	}
	return receipt, token, nil
}
