package expense

import (
	"errors"
	"expense-tracker/backend/services/ocr"
	"strings"
	"testing"
	"time"

	"expense-tracker/backend/types"

	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type receiptLifecycleTestStore struct {
	*mockExpenseStore
	receipt        types.ExpenseReceipt
	failCommitOnce bool
}

func (s *receiptLifecycleTestStore) RunInTransaction(callback func(types.ExpenseTransactionStore) error) error {
	copy := *s
	if err := callback(&copy); err != nil {
		return err
	}
	if s.failCommitOnce {
		s.failCommitOnce = false
		return errors.New("database commit failed")
	}
	s.receipt = copy.receipt
	return nil
}

func (s *receiptLifecycleTestStore) GetExpenseReceiptForUpdate(id uuid.UUID) (*types.ExpenseReceipt, error) {
	if id != s.receipt.ID {
		return nil, nil
	}
	copy := s.receipt
	return &copy, nil
}

func (s *receiptLifecycleTestStore) UpdateExpenseReceipt(receipt types.ExpenseReceipt) error {
	s.receipt = receipt
	return nil
}

func TestReceiptFinalizationRecoversFromCommitFailure(t *testing.T) {
	receipt, object := testReceiptObject()
	fake := &receiptS3Fake{objects: map[string]*s3.HeadObjectOutput{receipt.TemporaryKey: object}}
	store := &receiptLifecycleTestStore{mockExpenseStore: &mockExpenseStore{}, receipt: receipt, failCommitOnce: true}
	handler := &Handler{store: store, receiptObjects: &receiptObjectStore{client: fake, bucket: "private-receipts"}}

	_, err := handler.reconcileReceipt(t.Context(), receipt.ID)
	require.ErrorContains(t, err, "database commit failed")
	require.Equal(t, "pending", store.receipt.Status)
	require.Nil(t, fake.objects[receipt.TemporaryKey])
	require.NotNil(t, fake.objects[receipt.RetainedKey])

	result, err := handler.reconcileReceipt(t.Context(), receipt.ID)
	require.NoError(t, err)
	require.Equal(t, "finalized", result.Status)
	require.Equal(t, "finalized", store.receipt.Status)
	require.Equal(t, 1, fake.copies)
}

func TestReceiptRemovalRecoversFromCommitFailure(t *testing.T) {
	receipt, object := testReceiptObject()
	receipt.Role = "cleanup"
	receipt.Status = "deleting"
	fake := &receiptS3Fake{objects: map[string]*s3.HeadObjectOutput{receipt.RetainedKey: object}}
	store := &receiptLifecycleTestStore{mockExpenseStore: &mockExpenseStore{}, receipt: receipt, failCommitOnce: true}
	handler := &Handler{store: store, receiptObjects: &receiptObjectStore{client: fake, bucket: "private-receipts"}}

	_, err := handler.reconcileReceipt(t.Context(), receipt.ID)
	require.ErrorContains(t, err, "database commit failed")
	require.Equal(t, "deleting", store.receipt.Status)
	require.Empty(t, fake.objects)

	result, err := handler.reconcileReceipt(t.Context(), receipt.ID)
	require.NoError(t, err)
	require.Equal(t, "deleted", result.Status)
	require.Equal(t, "deleted", store.receipt.Status)
}

func TestPrepareReceiptRejectsAnotherAccountTokenBeforeS3(t *testing.T) {
	now := time.Now().UTC().Truncate(time.Second)
	objectID := uuid.NewString()
	key, err := ocr.TemporaryReceiptKey(objectID)
	require.NoError(t, err)
	secret := []byte(strings.Repeat("r", 32))
	token, err := ocr.MintReceiptToken(secret, ocr.TemporaryReceipt{
		AccountID: uuid.NewString(), ObjectID: objectID, Key: key,
		ChecksumSHA256: strings.Repeat("a", 64), ContentType: "image/jpeg",
		ByteSize: 24, Width: 4, Height: 6,
		CreatedAt: now, ExpiresAt: now.Add(ocr.TemporaryReceiptTTL),
	})
	require.NoError(t, err)
	fake := &receiptS3Fake{objects: map[string]*s3.HeadObjectOutput{}}
	handler := &Handler{receiptObjects: &receiptObjectStore{client: fake, bucket: "private-receipts"}, receiptSecret: secret, receiptStorageEnabled: true}
	_, err = handler.prepareReceipt(t.Context(), &types.ReceiptChoice{Keep: true, Token: token}, uuid.NewString(), uuid.New(), uuid.New())
	require.ErrorIs(t, err, errReceiptChoiceInvalid)
	require.Zero(t, fake.copies)
	require.Zero(t, fake.deletes)
}

func TestPrepareReceiptRequiresStorageFlag(t *testing.T) {
	fake := &receiptS3Fake{objects: map[string]*s3.HeadObjectOutput{}}
	handler := &Handler{
		receiptObjects: &receiptObjectStore{client: fake, bucket: "private-receipts"},
		receiptSecret:  []byte(strings.Repeat("r", 32)),
	}
	_, err := handler.prepareReceipt(t.Context(), &types.ReceiptChoice{Keep: true, Token: "token"}, uuid.NewString(), uuid.New(), uuid.New())
	require.ErrorIs(t, err, errReceiptUnavailable)
	require.Zero(t, fake.copies)
}
