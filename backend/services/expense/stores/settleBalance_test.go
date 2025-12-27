package store_test

import (
	"expense-tracker/backend/config"
	"expense-tracker/backend/db"
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestSettleBalanceByBalanceId(t *testing.T) {
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
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
			expectError:     nil,
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
			expectError:     nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			for _, balance := range test.mockBalance {
				insertBalance(db, &balance)
				defer deleteBalances(db, []*types.Balance{&balance})
			}

			err := store.SettleBalanceByBalanceId(test.mockBalanceId)

			updateBalanced := selectBalance(db, uuid.MustParse(test.mockBalanceId))

			if test.expectFail {
				assert.ErrorIs(t, err, test.expectError)
				assert.Equal(t, uuid.UUID{}, updateBalanced.ID)
			} else {
				assert.Nil(t, err)
				assert.Equal(t, test.mockBalanceId, updateBalanced.ID.String())
				assert.True(t, updateBalanced.IsSettled)
				assert.LessOrEqual(t,
					time.Since(updateBalanced.UpdateTime).Seconds(),
					time.Duration(1*time.Second).Seconds())
			}
		})
	}
}
