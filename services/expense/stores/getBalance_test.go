package store_test

import (
	"expense-tracker/config"
	"expense-tracker/db"
	expense "expense-tracker/services/expense/stores"
	"expense-tracker/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestGetBalanceByGroupId(t *testing.T) {
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	store := expense.NewStore(db)

	type testcase struct {
		name            string
		mockBalance     []types.Balance
		mockGroupId     string
		expectFail      bool
		expectResultLen int
		expectError     error
	}

	mockGroupID := uuid.New()
	mockGroupIDStr := mockGroupID.String()
	subtests := []testcase{
		{
			name: "valid",
			mockBalance: []types.Balance{
				{
					ID:             uuid.New(),
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
					GroupID:        mockGroupID},
			},
			mockGroupId:     mockGroupIDStr,
			expectFail:      false,
			expectResultLen: 1,
			expectError:     nil,
		},
		{
			name: "valid is_outdated",
			mockBalance: []types.Balance{
				{
					ID:             uuid.New(),
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
					IsOutdated:     true,
					UpdateTime:     time.Now(),
					IsSettled:      false,
				},
			},
			mockGroupId:     mockGroupIDStr,
			expectFail:      false,
			expectResultLen: 1,
			expectError:     nil,
		},
		{
			name: "valid is_settled",
			mockBalance: []types.Balance{
				{
					ID:             uuid.New(),
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
					IsOutdated:     false,
					IsSettled:      true,
					SettledTime:    time.Now(),
				},
			},
			mockGroupId:     mockGroupIDStr,
			expectFail:      false,
			expectResultLen: 1,
			expectError:     nil,
		},
		{
			name: "invalid nonexist group id",
			mockBalance: []types.Balance{
				{
					ID:             uuid.New(),
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
					GroupID:        mockGroupID,
				},
			},
			mockGroupId:     uuid.NewString(),
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

			balances, err := store.GetBalanceByGroupId(test.mockGroupId)

			if test.expectFail {
				assert.Len(t, balances, test.expectResultLen)
				assert.ErrorIs(t, err, test.expectError)
			} else {
				assert.NotNil(t, balances)
				assert.NotEmpty(t, balances)
				assert.NoError(t, err)
				assert.Len(t, balances, test.expectResultLen)
			}
		})
	}
}
