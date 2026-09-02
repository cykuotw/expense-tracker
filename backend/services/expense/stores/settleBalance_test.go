package store_test

import (
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestSettleBalanceByBalanceId(t *testing.T) {
	db := openTestDB(t)
	store := expense.NewStore(db)

	mockBalanceID := uuid.New()
	type testcase struct {
		name            string
		mockBalance     []types.Balance
		mockBalanceId   string
		expectFail      bool
		expectResultLen int
		expectError     error
	}

	subtests := []testcase{
		{
			name: "valid",
			mockBalance: []types.Balance{
				{
					ID:             mockBalanceID,
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
					GroupID:        mockGroupID,
				},
			},
			mockBalanceId:   mockBalanceID.String(),
			expectFail:      false,
			expectResultLen: 1,
			expectError:     types.ErrBalanceNotExist,
		},
		{
			name: "valid-2 records",
			mockBalance: []types.Balance{
				{
					ID:             mockBalanceID,
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
					GroupID:        mockGroupID,
				},
				{
					ID:             uuid.New(),
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
					GroupID:        mockGroupID,
				},
			},
			mockBalanceId:   mockBalanceID.String(),
			expectFail:      false,
			expectResultLen: 1,
			expectError:     nil,
		},
		{
			name: "invalid-unmatched id",
			mockBalance: []types.Balance{
				{
					ID:             mockBalanceID,
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
					GroupID:        mockGroupID,
				},
			},
			mockBalanceId:   uuid.NewString(),
			expectFail:      true,
			expectResultLen: 0,
			expectError:     types.ErrBalanceNotExist,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			for _, balance := range test.mockBalance {
				insertBalance(db, &balance)
				defer deleteBalances(db, []*types.Balance{&balance})
			}

			startedAt := time.Now().UTC().Add(-time.Second)
			err := store.SettleBalanceByBalanceId(mockGroupID.String(), test.mockBalanceId)
			finishedAt := time.Now().UTC().Add(time.Second)

			updateBalanced := selectBalance(db, uuid.MustParse(test.mockBalanceId))

			if test.expectFail {
				assert.ErrorIs(t, err, test.expectError)
				assert.Equal(t, uuid.UUID{}, updateBalanced.ID)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.mockBalanceId, updateBalanced.ID.String())
				assert.True(t, updateBalanced.IsSettled)
				assert.False(t, updateBalanced.UpdateTime.IsZero())
				assert.False(t, updateBalanced.UpdateTime.Before(startedAt))
				assert.False(t, updateBalanced.UpdateTime.After(finishedAt))
				assert.True(t, updateBalanced.UpdateTime.Equal(updateBalanced.SettledTime))
			}
		})
	}
}
