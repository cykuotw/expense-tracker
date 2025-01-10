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

func TestCheckGroupBallanceAllSettled(t *testing.T) {
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	store := expense.NewStore(db)

	type testcase struct {
		name         string
		mockBalance  []*types.Balance
		mockGroupId  string
		expectFail   bool
		expectResult bool
		expectError  error
	}

	subtests := []testcase{
		{
			name: "valid-1 settled",
			mockBalance: []*types.Balance{
				{
					ID:             uuid.New(),
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
					GroupID:        mockGroupID,
					IsSettled:      true,
					SettledTime:    time.Now(),
				},
			},
			mockGroupId:  mockGroupID.String(),
			expectFail:   false,
			expectResult: true,
			expectError:  nil,
		},
		{
			name: "valid-2 settled",
			mockBalance: []*types.Balance{
				{
					ID:             uuid.New(),
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
					GroupID:        mockGroupID,
					IsSettled:      true,
					SettledTime:    time.Now(),
				},
				{
					ID:             uuid.New(),
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
					GroupID:        mockGroupID,
					IsSettled:      true,
					SettledTime:    time.Now(),
				},
			},
			mockGroupId:  mockGroupID.String(),
			expectFail:   false,
			expectResult: true,
			expectError:  nil,
		},
		{
			name: "valid-1 settled 1 unsettled",
			mockBalance: []*types.Balance{
				{
					ID:             uuid.New(),
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
					GroupID:        mockGroupID,
					IsSettled:      true,
					SettledTime:    time.Now(),
				},
				{
					ID:             uuid.New(),
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
					GroupID:        mockGroupID,
					IsSettled:      false,
				},
			},
			mockGroupId:  mockGroupID.String(),
			expectFail:   false,
			expectResult: false,
			expectError:  nil,
		},
		{
			name: "valid-1 unsettled",
			mockBalance: []*types.Balance{
				{
					ID:             uuid.New(),
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
					GroupID:        mockGroupID,
					IsSettled:      false,
					SettledTime:    time.Now(),
				},
			},
			mockGroupId:  mockGroupID.String(),
			expectFail:   false,
			expectResult: false,
			expectError:  nil,
		},
		{
			name:         "valid-empty group",
			mockBalance:  []*types.Balance{},
			mockGroupId:  mockGroupID.String(),
			expectFail:   false,
			expectResult: true,
			expectError:  nil,
		},
		{
			name: "valid-unmatched group",
			mockBalance: []*types.Balance{
				{
					ID:             uuid.New(),
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
					GroupID:        mockGroupID,
					IsSettled:      false,
					SettledTime:    time.Now(),
				},
			},
			mockGroupId:  uuid.NewString(),
			expectFail:   false,
			expectResult: true,
			expectError:  nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			for _, bal := range test.mockBalance {
				insertBalance(db, bal)
			}
			defer deleteBalances(db, test.mockBalance)

			exist, err := store.CheckGroupBallanceAllSettled(test.mockGroupId)

			if !test.expectFail {
				assert.Nil(t, err)
				assert.Equal(t, test.expectResult, exist)
			}
		})
	}
}
