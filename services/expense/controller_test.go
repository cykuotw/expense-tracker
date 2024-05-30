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

	ledgers := []*types.Ledger{
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
	}

	// sender: balance
	expectBalance := []*types.Balance{
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
	}

	t.Run("valid", func(t *testing.T) {
		balanceList := expense.DebtSimplify(ledgers)

		assert.Equal(t, len(expectBalance), len(balanceList))

		for _, b := range balanceList {
			expectBalance := findBalance(expectBalance, b.SenderUserID, b.ReceiverUserID)

			assert.NotNil(t, expectBalance)
			assert.True(t, expectBalance.Share.Equal(b.Share))
		}

		// make sure Alice does not owe or be owed by anyone
		var balance *types.Balance
		for _, b := range expectBalance {
			if b.SenderUserID == AliceID || b.ReceiverUserID == AliceID {
				balance = b
				break
			}
		}
		assert.Nil(t, balance)
	})
}

func findBalance(expectBalance []*types.Balance, senderID uuid.UUID, receiverID uuid.UUID) *types.Balance {
	var result *types.Balance

	for _, b := range expectBalance {
		if b.SenderUserID == senderID && b.ReceiverUserID == receiverID {
			result = b
			break
		}
	}

	return result
}
