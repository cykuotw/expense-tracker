package expense_test

import (
	"expense-tracker/services/expense"
	"expense-tracker/types"
	"testing"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestDebtSimplify(t *testing.T) {
	AliceID := uuid.New() // owes and is owed by no one
	GabeID := uuid.New()
	FredID := uuid.New()
	BobID := uuid.New()
	CharlieID := uuid.New()
	DavidID := uuid.New()
	EmaID := uuid.New()

	type testcase struct {
		name          string
		ledgers       []*types.Ledger
		expectFail    bool
		expectBalance []*types.Balance
	}

	subtests := []testcase{
		{
			name: "valid 1",
			ledgers: []*types.Ledger{
				// Gabe owes Bob $30
				{
					LenderUserID:   BobID,
					BorrowerUesrID: GabeID,
					Share:          decimal.NewFromInt(30),
				},
				// Gabe owes David $10
				{
					LenderUserID:   DavidID,
					BorrowerUesrID: GabeID,
					Share:          decimal.NewFromInt(10),
				},
				// Fred owes Bob $10
				{
					LenderUserID:   BobID,
					BorrowerUesrID: FredID,
					Share:          decimal.NewFromInt(10),
				},
				// Fred owes Charlie $30
				{
					LenderUserID:   CharlieID,
					BorrowerUesrID: FredID,
					Share:          decimal.NewFromInt(30),
				},
				// Fred owes David $10
				{
					LenderUserID:   DavidID,
					BorrowerUesrID: FredID,
					Share:          decimal.NewFromInt(10),
				},
				// Fred owes Ema $10
				{
					LenderUserID:   EmaID,
					BorrowerUesrID: FredID,
					Share:          decimal.NewFromInt(10),
				},
				// Bob owes Charlie $40
				{
					LenderUserID:   CharlieID,
					BorrowerUesrID: BobID,
					Share:          decimal.NewFromInt(40),
				},
				// Charlie owes David $20
				{
					LenderUserID:   DavidID,
					BorrowerUesrID: CharlieID,
					Share:          decimal.NewFromInt(20),
				},
				// David owes Ema $50
				{
					LenderUserID:   EmaID,
					BorrowerUesrID: DavidID,
					Share:          decimal.NewFromInt(50),
				},
			},
			expectFail: false,
			expectBalance: []*types.Balance{
				// all possible combinations
				// Ema send Fred $60
				{
					SenderUserID:   EmaID,
					ReceiverUserID: FredID,
					Share:          decimal.NewFromInt(60),
				},
				// Charlie send Gabe $40
				{
					SenderUserID:   CharlieID,
					ReceiverUserID: GabeID,
					Share:          decimal.NewFromInt(40),
				},
				// Gabe send David $20
				{
					SenderUserID:   CharlieID,
					ReceiverUserID: DavidID,
					Share:          decimal.NewFromInt(10),
				},
				// Gabe send David $50
				{
					SenderUserID:   CharlieID,
					ReceiverUserID: GabeID,
					Share:          decimal.NewFromInt(50),
				},
				// Gabe send David $10
				{
					SenderUserID:   GabeID,
					ReceiverUserID: DavidID,
					Share:          decimal.NewFromInt(10),
				},
			},
		},
		{
			name: "valie 2",
			ledgers: []*types.Ledger{
				{
					LenderUserID:   AliceID,
					BorrowerUesrID: BobID,
					Share:          decimal.NewFromFloat(92.0),
				},
				{
					LenderUserID:   BobID,
					BorrowerUesrID: AliceID,
					Share:          decimal.NewFromFloat(1.24),
				},
				{
					LenderUserID:   BobID,
					BorrowerUesrID: AliceID,
					Share:          decimal.NewFromFloat(7.61),
				},
				{
					LenderUserID:   BobID,
					BorrowerUesrID: AliceID,
					Share:          decimal.NewFromFloat(41.4),
				},
				{
					LenderUserID:   AliceID,
					BorrowerUesrID: BobID,
					Share:          decimal.NewFromFloat(5.1),
				},
				{
					LenderUserID:   BobID,
					BorrowerUesrID: AliceID,
					Share:          decimal.NewFromFloat(73),
				},
				{
					LenderUserID:   AliceID,
					BorrowerUesrID: BobID,
					Share:          decimal.NewFromFloat(7.22),
				},
			},
			expectFail: false,
			expectBalance: []*types.Balance{
				{
					SenderUserID:   BobID,
					ReceiverUserID: AliceID,
					Share:          decimal.NewFromFloat(18.93),
				},
			},
		},
	}

	// first test case
	t.Run(subtests[0].name, func(t *testing.T) {
		balanceList := expense.DebtSimplify(subtests[0].ledgers)

		for _, b := range balanceList {
			matchResult := matchBalance(subtests[0].expectBalance, b)

			assert.True(t, matchResult)
		}

		// make sure Alice does not owe or be owed by anyone
		var balance *types.Balance
		for _, b := range subtests[0].expectBalance {
			if b.SenderUserID == AliceID || b.ReceiverUserID == AliceID {
				balance = b
				break
			}
		}
		assert.Nil(t, balance)
	})

	// second test case
	t.Run(subtests[1].name, func(t *testing.T) {
		balanceList := expense.DebtSimplify(subtests[0].ledgers)

		for _, b := range balanceList {
			matchResult := matchBalance(subtests[0].expectBalance, b)

			assert.True(t, matchResult)
		}
	})
}

func matchBalance(expectBalance []*types.Balance, balance *types.Balance) bool {
	result := false

	for _, b := range expectBalance {
		if b.SenderUserID == balance.SenderUserID &&
			b.ReceiverUserID == balance.ReceiverUserID &&
			b.Share.Equal(balance.Share) {
			result = true
			break
		}
	}

	return result
}
