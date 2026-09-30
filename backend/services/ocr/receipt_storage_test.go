package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/aws/aws-sdk-go-v2/service/s3/types"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type receiptPutRecorder struct {
	input *s3.PutObjectInput
	body  []byte
	calls int
	err   error
}

func (recorder *receiptPutRecorder) PutObject(_ context.Context, input *s3.PutObjectInput, _ ...func(*s3.Options)) (*s3.PutObjectOutput, error) {
	recorder.calls++
	recorder.input = input
	body, err := io.ReadAll(input.Body)
	if err != nil {
		return nil, err
	}
	recorder.body = body
	if recorder.err != nil {
		return nil, recorder.err
	}
	return &s3.PutObjectOutput{}, nil
}

func TestTemporaryReceiptStoreWritesOnlyNormalizedPixelsAndMetadata(t *testing.T) {
	now := time.Date(2026, time.September, 30, 12, 0, 0, 0, time.UTC)
	secret := []byte(strings.Repeat("s", 32))
	accountID := uuid.NewString()
	source := encodeTestJPEG(t, 8, 6)
	marker := []byte("Exif\x00\x00SENSITIVE_GPS_METADATA")
	length := len(marker) + 2
	withMetadata := append(append(append([]byte{}, source[:2]...), 0xff, 0xe1, byte(length>>8), byte(length)), marker...)
	withMetadata = append(withMetadata, source[2:]...)
	processed, err := PreprocessRaster(bytes.NewReader(withMetadata), DefaultPreprocessOptions())
	require.NoError(t, err)

	recorder := &receiptPutRecorder{}
	store, err := NewTemporaryReceiptStore(recorder, "private-receipts", secret)
	require.NoError(t, err)
	receipt, token, err := store.Store(t.Context(), accountID, processed, now)
	require.NoError(t, err)
	assert.Equal(t, 1, recorder.calls)
	assert.Equal(t, processed.Bytes, recorder.body)
	assert.NotContains(t, string(recorder.body), "SENSITIVE_GPS_METADATA")
	assert.Equal(t, "private-receipts", aws.ToString(recorder.input.Bucket))
	assert.Equal(t, receipt.Key, aws.ToString(recorder.input.Key))
	assert.Regexp(t, `^temporary/[0-9a-f-]{36}\.jpg$`, receipt.Key)
	assert.Equal(t, "*", aws.ToString(recorder.input.IfNoneMatch))
	assert.Equal(t, types.ServerSideEncryptionAes256, recorder.input.ServerSideEncryption)
	assert.Equal(t, "temporary", recorder.input.Metadata["lifecycle-state"])
	assert.Equal(t, receipt.ChecksumSHA256, recorder.input.Metadata["checksum-sha256"])
	assert.Len(t, recorder.input.Metadata, 8)
	verified, err := VerifyReceiptToken(secret, token, now.Add(time.Minute), accountID)
	require.NoError(t, err)
	assert.Equal(t, receipt.Key, verified.Key)
	assert.Equal(t, receipt.ChecksumSHA256, verified.ChecksumSHA256)
	assert.Equal(t, receipt.AccountID, verified.AccountID)
	assert.True(t, receipt.CreatedAt.Equal(verified.CreatedAt))
	assert.True(t, receipt.ExpiresAt.Equal(verified.ExpiresAt))
}

func TestTemporaryReceiptStoreRejectsUnvalidatedInputAndFailedWrites(t *testing.T) {
	recorder := &receiptPutRecorder{}
	store, err := NewTemporaryReceiptStore(recorder, "private-receipts", []byte(strings.Repeat("s", 32)))
	require.NoError(t, err)
	_, token, err := store.Store(t.Context(), uuid.NewString(), PreprocessedImage{Bytes: encodeTestJPEG(t, 8, 8), Width: 8, Height: 8}, time.Now())
	assert.ErrorIs(t, err, ErrInvalidTemporaryReceipt)
	assert.Empty(t, token)
	assert.Zero(t, recorder.calls)

	processed, err := PreprocessRaster(bytes.NewReader(encodeTestJPEG(t, 8, 8)), DefaultPreprocessOptions())
	require.NoError(t, err)
	recorder.err = errors.New("S3 unavailable")
	_, token, err = store.Store(t.Context(), uuid.NewString(), processed, time.Now())
	assert.Error(t, err)
	assert.Empty(t, token)
	assert.Equal(t, 1, recorder.calls)
}

func TestReceiptTokenRejectsForgedExpiredWrongAccountAndModified(t *testing.T) {
	now := time.Date(2026, time.September, 30, 12, 0, 0, 0, time.UTC)
	secret := []byte(strings.Repeat("s", 32))
	accountID := uuid.NewString()
	processed, err := PreprocessRaster(bytes.NewReader(encodeTestJPEG(t, 8, 8)), DefaultPreprocessOptions())
	require.NoError(t, err)
	store, err := NewTemporaryReceiptStore(&receiptPutRecorder{}, "private-receipts", secret)
	require.NoError(t, err)
	_, token, err := store.Store(t.Context(), accountID, processed, now)
	require.NoError(t, err)
	parts := strings.Split(token, ".")
	require.Len(t, parts, 3)
	parts[1] = "X" + parts[1][1:]
	modified := strings.Join(parts, ".")
	ocrToken, _, err := MintCapability(secret, now, accountID, uuid.NewString(), "image/jpeg")
	require.NoError(t, err)

	for name, test := range map[string]struct {
		secret  []byte
		token   string
		now     time.Time
		account string
	}{
		"forged":              {[]byte(strings.Repeat("f", 32)), token, now.Add(time.Minute), accountID},
		"expired":             {secret, token, now.Add(ReceiptTokenTTL + time.Second), accountID},
		"wrong account":       {secret, token, now.Add(time.Minute), uuid.NewString()},
		"modified":            {secret, modified, now.Add(time.Minute), accountID},
		"wrong token purpose": {secret, ocrToken, now.Add(time.Minute), accountID},
	} {
		t.Run(name, func(t *testing.T) {
			_, err := VerifyReceiptToken(test.secret, test.token, test.now, test.account)
			assert.ErrorIs(t, err, ErrInvalidReceiptToken)
		})
	}
}

func TestDraftStorageRequiresBothExplicitIntentAndRuntimeEnablement(t *testing.T) {
	now, accountID, requestID, _, secret := draftFixture(t, "image/jpeg")
	recorder := &receiptPutRecorder{}
	store, err := NewTemporaryReceiptStore(recorder, "private-receipts", secret)
	require.NoError(t, err)
	image := encodeTestJPEG(t, 8, 8)
	for name, test := range map[string]struct {
		keepReceipt bool
		enabled     bool
		expectWrite bool
	}{
		"normal scan":           {false, true, false},
		"disabled runtime":      {true, false, false},
		"explicit enabled scan": {true, true, true},
	} {
		t.Run(name, func(t *testing.T) {
			recorder.calls = 0
			token, _, err := MintCapability(secret, now, accountID, requestID, "image/jpeg")
			if test.keepReceipt {
				token, _, err = MintReceiptRetentionCapability(secret, now, accountID, requestID, "image/jpeg")
			}
			require.NoError(t, err)
			handler := NewDraftHandler(RuntimeConfig{
				FrontendOrigin: "https://app.example.com", CapabilitySecret: secret,
				ReceiptStorageEnabled: test.enabled,
			}, &replayStub{}, &providerStub{}, nil)
			handler.now = func() time.Time { return now.Add(time.Second) }
			handler.SetTemporaryReceiptStore(store)
			response, err := handler.Handle(t.Context(), stubRequest(token, accountID, requestID, "image/jpeg", image))
			require.NoError(t, err)
			require.Equal(t, http.StatusOK, response.StatusCode)
			var payload DraftResponse
			require.NoError(t, json.Unmarshal([]byte(response.Body), &payload))
			if test.expectWrite {
				assert.Equal(t, 1, recorder.calls)
				assert.NotEmpty(t, payload.ReceiptToken)
			} else {
				assert.Zero(t, recorder.calls)
				assert.Empty(t, payload.ReceiptToken)
			}
		})
	}
}

func TestDraftStorageFailureReturnsRetryableErrorWithoutReceiptToken(t *testing.T) {
	now, accountID, requestID, _, secret := draftFixture(t, "image/jpeg")
	token, _, err := MintReceiptRetentionCapability(secret, now, accountID, requestID, "image/jpeg")
	require.NoError(t, err)
	recorder := &receiptPutRecorder{err: errors.New("internal storage detail")}
	store, err := NewTemporaryReceiptStore(recorder, "private-receipts", secret)
	require.NoError(t, err)
	var observation DraftObservation
	handler := NewDraftHandler(RuntimeConfig{
		FrontendOrigin: "https://app.example.com", CapabilitySecret: secret,
		ReceiptStorageEnabled: true,
	}, &replayStub{}, &providerStub{}, func(value DraftObservation) { observation = value })
	handler.now = func() time.Time { return now.Add(time.Second) }
	handler.SetTemporaryReceiptStore(store)
	response, err := handler.Handle(t.Context(), stubRequest(token, accountID, requestID, "image/jpeg", encodeTestJPEG(t, 8, 8)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	assert.Equal(t, "1", response.Headers["Retry-After"])
	assert.NotContains(t, response.Body, "internal storage detail")
	assert.NotContains(t, response.Body, "receiptToken")
	assert.Equal(t, "failed", observation.ReceiptStorageOutcome)
	assert.Equal(t, "receipt_storage_error", observation.Outcome)
}
