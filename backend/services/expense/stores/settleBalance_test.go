package store_test

import (
	"database/sql"
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSettleBalanceByBalanceID(t *testing.T) {
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
			err := store.SettleBalanceByBalanceID(mockGroupID.String(), test.mockBalanceId, test.mockBalance[0].SenderUserID)
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

func TestSettleBalanceByBalanceIDAuthorizationAndIdempotency(t *testing.T) {
	db := openTestDB(t)
	store := expense.NewStore(db)

	t.Run("non-party cannot settle current balance", func(t *testing.T) {
		balance := &types.Balance{
			ID:             uuid.New(),
			SenderUserID:   uuid.New(),
			ReceiverUserID: uuid.New(),
			Share:          decimal.NewFromInt(20),
			GroupID:        uuid.New(),
		}
		require.NoError(t, insertBalance(db, balance))
		cleanupSettlementBalance(t, db, balance)

		err := store.SettleBalanceByBalanceID(balance.GroupID.String(), balance.ID.String(), uuid.New())

		require.ErrorIs(t, err, types.ErrUserNotPermitted)
		assert.False(t, selectBalance(db, balance.ID).IsSettled)
	})

	t.Run("outdated balance is not a settlement target", func(t *testing.T) {
		balance := &types.Balance{
			ID:             uuid.New(),
			SenderUserID:   uuid.New(),
			ReceiverUserID: uuid.New(),
			Share:          decimal.NewFromInt(20),
			GroupID:        uuid.New(),
			IsOutdated:     true,
		}
		require.NoError(t, insertBalance(db, balance))
		cleanupSettlementBalance(t, db, balance)

		err := store.SettleBalanceByBalanceID(balance.GroupID.String(), balance.ID.String(), balance.SenderUserID)

		require.ErrorIs(t, err, types.ErrBalanceNotExist)
		assert.False(t, selectBalance(db, balance.ID).IsSettled)
	})

	t.Run("already-settled current balance is an unchanged success", func(t *testing.T) {
		settledAt := time.Now().UTC().Add(-time.Hour).Truncate(time.Microsecond)
		balance := &types.Balance{
			ID:             uuid.New(),
			SenderUserID:   uuid.New(),
			ReceiverUserID: uuid.New(),
			Share:          decimal.NewFromInt(20),
			GroupID:        uuid.New(),
			IsSettled:      true,
			UpdateTime:     settledAt,
			SettledTime:    settledAt,
		}
		require.NoError(t, insertBalance(db, balance))
		cleanupSettlementBalance(t, db, balance)

		require.NoError(t, store.SettleBalanceByBalanceID(balance.GroupID.String(), balance.ID.String(), balance.ReceiverUserID))

		updated := selectBalance(db, balance.ID)
		assert.True(t, updated.SettledTime.Equal(settledAt.Truncate(time.Second)))
		assert.True(t, updated.UpdateTime.Equal(settledAt.Truncate(time.Second)))
	})
}

func cleanupSettlementBalance(t *testing.T, db *sql.DB, balance *types.Balance) {
	t.Helper()
	t.Cleanup(func() {
		deleteBalances(db, []*types.Balance{balance})
		_, err := db.Exec("DELETE FROM groups WHERE id = $1", balance.GroupID)
		require.NoError(t, err)
		_, err = db.Exec("DELETE FROM users WHERE id = ANY($1)", []uuid.UUID{balance.SenderUserID, balance.ReceiverUserID})
		require.NoError(t, err)
	})
}
