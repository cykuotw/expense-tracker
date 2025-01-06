package store_test

import (
	"database/sql"
	"expense-tracker/config"
	"expense-tracker/db"
	expense "expense-tracker/services/expense/stores"
	"expense-tracker/types"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestOutdateBalanceByGroupId(t *testing.T) {
	// prepare test data
	cfg := config.Envs
	db, _ := db.NewPostgreSQLStorage(cfg)
	store := expense.NewStore(db)

	type testcase struct {
		name          string
		mockBalance   types.Balance
		mockBalanceId uuid.UUID
		mockGroupId   uuid.UUID
		expectFail    bool
		expectError   error
	}

	subtests := []testcase{
		{
			name: "valid",
			mockBalance: types.Balance{
				ID:             uuid.New(),
				SenderUserID:   uuid.New(),
				ReceiverUserID: uuid.New(),
				Share:          decimal.NewFromFloat(20.01),
			},
			mockGroupId: uuid.New(),
			expectFail:  false,
			expectError: nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			test.mockBalance.GroupID = test.mockGroupId
			insertBalance(db, &test.mockBalance)
			defer deleteBalances(db, []*types.Balance{&test.mockBalance})

			err := store.OutdateBalanceByGroupId(test.mockGroupId.String())

			if !test.expectFail {
				assert.Nil(t, err)

				balance := selectBalance(db, test.mockBalance.ID)

				assert.Equal(t, test.mockBalance.ID, balance.ID)
				assert.True(t, balance.IsOutdated)
				assert.LessOrEqual(t,
					balance.UpdateTime.Sub(time.Now()).Seconds(),
					time.Duration(1*time.Second).Seconds())
			}
		})
	}
}

func insertBalance(db *sql.DB, balance *types.Balance) {
	createTime := balance.CreateTime.UTC().Format("2006-01-02 15:04:05-0700")
	updateTime := balance.UpdateTime.UTC().Format("2006-01-02 15:04:05-0700")
	settleTime := balance.SettledTime.UTC().Format("2006-01-02 15:04:05-0700")
	query := fmt.Sprintf(
		"INSERT INTO balance ("+
			"id, sender_user_id, receiver_user_id, share, group_id, "+
			"create_time_utc, is_outdated, update_time_utc, "+
			"is_settled, settle_time_utc"+
			") VALUES ("+
			"'%s', '%s', '%s', '%s', '%s', "+
			"'%s', '%t', '%s', '%t', '%s'"+
			")",
		balance.ID,
		balance.SenderUserID, balance.ReceiverUserID, balance.Share.String(),
		balance.GroupID,
		createTime, balance.IsOutdated, updateTime,
		balance.IsSettled, settleTime,
	)

	db.Exec(query)
}
