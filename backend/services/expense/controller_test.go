package expense_test

import (
	"expense-tracker/backend/services/expense"
	"expense-tracker/backend/types"
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
		name       string
		ledgers    []*types.Ledger
		expectFail bool
	}

	controller := expense.NewController()

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
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			balanceList := controller.DebtSimplify(test.ledgers)

			expected := netFromLedgers(test.ledgers)
			actual := netFromBalances(balanceList)

			for id, exp := range expected {
				act := actual[id]
				assert.True(t, exp.Round(2).Equal(act.Round(2)))
			}
			for id, act := range actual {
				if _, ok := expected[id]; !ok {
					assert.True(t, act.Round(2).IsZero())
				}
			}

			sumActual := decimal.Zero
			for _, act := range actual {
				sumActual = sumActual.Add(act)
			}
			assert.True(t, sumActual.Round(2).IsZero())
		})
	}
}

func BenchmarkDebtSimplify(b *testing.B) {
	GabeID := uuid.New()
	FredID := uuid.New()
	BobID := uuid.New()
	CharlieID := uuid.New()
	DavidID := uuid.New()
	EmaID := uuid.New()

	controller := expense.NewController()

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

	for i := 0; i < b.N; i++ {
		controller.DebtSimplify(ledgers)
	}
}

func netFromLedgers(ledgers []*types.Ledger) map[uuid.UUID]decimal.Decimal {
	net := map[uuid.UUID]decimal.Decimal{}
	for _, l := range ledgers {
		if v, ok := net[l.LenderUserID]; ok {
			net[l.LenderUserID] = v.Sub(l.Share)
		} else {
			net[l.LenderUserID] = l.Share.Neg()
		}
		if v, ok := net[l.BorrowerUesrID]; ok {
			net[l.BorrowerUesrID] = v.Add(l.Share)
		} else {
			net[l.BorrowerUesrID] = l.Share
		}
	}
	return net
}

func netFromBalances(balances []*types.Balance) map[uuid.UUID]decimal.Decimal {
	net := map[uuid.UUID]decimal.Decimal{}
	for _, b := range balances {
		if v, ok := net[b.SenderUserID]; ok {
			net[b.SenderUserID] = v.Add(b.Share)
		} else {
			net[b.SenderUserID] = b.Share
		}
		if v, ok := net[b.ReceiverUserID]; ok {
			net[b.ReceiverUserID] = v.Sub(b.Share)
		} else {
			net[b.ReceiverUserID] = b.Share.Neg()
		}
	}
	return net
}
