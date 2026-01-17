package store_test

import (
	"database/sql"
	expense "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/types"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestCreateBalance(t *testing.T) {
	// prepare test data
	db := openTestDB(t)
	store := expense.NewStore(db)

	type testcase struct {
		name         string
		mockBalances []*types.Balance
		mockGroupId  string
		expectFail   bool
		expectError  error
	}

	subtests := []testcase{
		{
			name: "valid",
			mockBalances: []*types.Balance{
				{
					ID:             uuid.New(),
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
				},
				{
					ID:             uuid.New(),
					SenderUserID:   uuid.New(),
					ReceiverUserID: uuid.New(),
					Share:          decimal.NewFromFloat(20.01),
				},
			},
			mockGroupId: uuid.New().String(),
			expectFail:  false,
			expectError: nil,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			err := store.CreateBalances(test.mockGroupId, test.mockBalances)
			defer deleteBalances(db, test.mockBalances)

			if !test.expectFail {
				assert.Nil(t, err)
				for _, bal := range test.mockBalances {
					exist := checkBalanceExist(db, bal.ID)
					assert.True(t, exist)
				}

			}
		})
	}

}

func deleteBalances(db *sql.DB, balances []*types.Balance) {
	for _, balance := range balances {
		query := fmt.Sprintf(`DELETE FROM balance WHERE id = '%s';`, balance.ID)
		db.Exec(query)
	}
}

func checkBalanceExist(db *sql.DB, balanceId uuid.UUID) bool {
	query := fmt.Sprintf("SELECT EXISTS (SELECT 1 FROM balance WHERE id = '%s')", balanceId)
	rows, _ := db.Query(query)
	defer rows.Close()

	exist := false
	for rows.Next() {
		err := rows.Scan(&exist)
		if err != nil {
			return false
		}
	}

	return exist
}

func selectBalance(db *sql.DB, balanceId uuid.UUID) types.Balance {
	query := fmt.Sprintf("SELECT * FROM balance WHERE id='%s';", balanceId)
	rows, _ := db.Query(query)
	defer rows.Close()

	var balance types.Balance
	for rows.Next() {
		updateTime := new(time.Time)
		settledTime := new(time.Time)
		rows.Scan(
			&balance.ID,
			&balance.SenderUserID,
			&balance.ReceiverUserID,
			&balance.Share,
			&balance.GroupID,
			&balance.CreateTime,
			&balance.IsOutdated,
			updateTime,
			&balance.IsSettled,
			settledTime,
		)
		if !updateTime.IsZero() {
			balance.UpdateTime = *updateTime
		}
		if !settledTime.IsZero() {
			balance.SettledTime = *settledTime
		}
	}

	return balance
}
