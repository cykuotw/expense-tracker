package allocation

import (
	"cmp"
	"slices"

	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var maximumMoneyValue = decimal.NewFromInt(10_000_000).Sub(decimal.New(1, -3))
var maximumItemQuantity = decimal.NewFromInt(100_000_000).Sub(decimal.New(1, -2))

// ParsePayload validates and normalizes an allocation request into stable user order.
func ParsePayload(expenseID uuid.UUID, payload types.ExpenseAllocationPayload) ([]types.ExpenseAllocation, error) {
	if len(payload.Participants) == 0 {
		return nil, types.ErrInvalidMoney
	}
	allocations := make([]types.ExpenseAllocation, 0, len(payload.Participants))
	seen := make(map[uuid.UUID]struct{}, len(payload.Participants))
	for _, participant := range payload.Participants {
		userID, err := uuid.Parse(participant.UserID)
		if err != nil {
			return nil, types.ErrInvalidMoney
		}
		if _, exists := seen[userID]; exists {
			return nil, types.ErrInvalidMoney
		}
		seen[userID] = struct{}{}
		allocation := types.ExpenseAllocation{
			ExpenseID: expenseID, UserID: userID, Amount: participant.Amount,
			PercentageBasisPoints: participant.PercentageBasisPoints,
		}
		switch payload.Mode {
		case types.ExpenseAllocationEqual:
			if allocation.Amount != nil || allocation.PercentageBasisPoints != nil {
				return nil, types.ErrInvalidMoney
			}
		case types.ExpenseAllocationExact:
			if allocation.Amount == nil || allocation.PercentageBasisPoints != nil {
				return nil, types.ErrInvalidMoney
			}
		case types.ExpenseAllocationPercentage:
			if allocation.Amount != nil || allocation.PercentageBasisPoints == nil {
				return nil, types.ErrInvalidMoney
			}
		case types.ExpenseAllocationAdjustment:
			if allocation.PercentageBasisPoints != nil {
				return nil, types.ErrInvalidMoney
			}
			if allocation.Amount == nil {
				zero := decimal.Zero
				allocation.Amount = &zero
			}
		default:
			return nil, types.ErrInvalidAction
		}
		allocations = append(allocations, allocation)
	}
	slices.SortFunc(allocations, func(a, b types.ExpenseAllocation) int {
		return cmp.Compare(a.UserID.String(), b.UserID.String())
	})
	return allocations, nil
}

// ValidateMoney verifies expense totals and item precision for a currency.
func ValidateMoney(expense types.Expense, items []types.Item, amountDigits int32) error {
	if expense.Total.LessThanOrEqual(decimal.Zero) || !validMoney(expense.Total, amountDigits) ||
		!validMoney(expense.SubTotal, amountDigits) || !validMoney(expense.TaxFeeTip, amountDigits) {
		return types.ErrInvalidMoney
	}
	itemSubtotal := decimal.Zero
	for _, item := range items {
		if item.Amount.LessThanOrEqual(decimal.Zero) ||
			item.Amount.Abs().GreaterThan(maximumItemQuantity) ||
			!item.Amount.Truncate(2).Equal(item.Amount) || item.UnitPrice.IsNegative() ||
			!validMoney(item.UnitPrice, amountDigits) {
			return types.ErrInvalidMoney
		}
		itemSubtotal = itemSubtotal.Add(item.Amount.Mul(item.UnitPrice).Round(amountDigits))
	}
	if len(items) > 0 && !itemSubtotal.Equal(expense.SubTotal) {
		return types.ErrInvalidMoney
	}
	return nil
}

// DeriveLedgers deterministically converts a validated allocation into ledger shares.
func DeriveLedgers(expense types.Expense, allocations []types.ExpenseAllocation, amountDigits int32) ([]types.Ledger, error) {
	if len(allocations) == 0 {
		return nil, types.ErrInvalidMoney
	}
	totalUnits := expense.Total.Shift(amountDigits).IntPart()
	shares := make([]int64, len(allocations))
	switch expense.AllocationMode {
	case types.ExpenseAllocationEqual:
		allocateEqualUnits(totalUnits, shares)
	case types.ExpenseAllocationExact:
		var sum int64
		for index, allocation := range allocations {
			if allocation.Amount == nil || allocation.PercentageBasisPoints != nil ||
				allocation.Amount.IsNegative() || !validMoney(*allocation.Amount, amountDigits) {
				return nil, types.ErrInvalidMoney
			}
			shares[index] = allocation.Amount.Shift(amountDigits).IntPart()
			sum += shares[index]
		}
		if sum != totalUnits {
			return nil, types.ErrInvalidMoney
		}
	case types.ExpenseAllocationPercentage:
		if err := allocatePercentageUnits(totalUnits, allocations, shares); err != nil {
			return nil, err
		}
	case types.ExpenseAllocationAdjustment:
		var adjustmentTotal int64
		for index, allocation := range allocations {
			if allocation.Amount == nil || allocation.PercentageBasisPoints != nil ||
				allocation.Amount.IsNegative() || !validMoney(*allocation.Amount, amountDigits) {
				return nil, types.ErrInvalidMoney
			}
			shares[index] = allocation.Amount.Shift(amountDigits).IntPart()
			adjustmentTotal += shares[index]
		}
		if adjustmentTotal > totalUnits {
			return nil, types.ErrInvalidMoney
		}
		baseShares := make([]int64, len(allocations))
		allocateEqualUnits(totalUnits-adjustmentTotal, baseShares)
		for index := range shares {
			shares[index] += baseShares[index]
		}
	default:
		return nil, types.ErrInvalidAction
	}
	ledgers := make([]types.Ledger, 0, len(allocations))
	for index, allocation := range allocations {
		ledgers = append(ledgers, types.Ledger{
			ID: uuid.New(), ExpenseID: expense.ID, LenderUserID: expense.PayByUserId,
			BorrowerUesrID: allocation.UserID,
			Share:          decimal.NewFromInt(shares[index]).Shift(-amountDigits),
		})
	}
	return ledgers, nil
}

// ParticipantIDs returns each actor involved in the mutation exactly once.
func ParticipantIDs(actorID uuid.UUID, expense types.Expense, allocations []types.ExpenseAllocation) []uuid.UUID {
	userIDs := make([]uuid.UUID, 0, 2+len(allocations))
	seen := make(map[uuid.UUID]struct{}, cap(userIDs))
	appendUnique := func(userID uuid.UUID) {
		if _, exists := seen[userID]; exists {
			return
		}
		seen[userID] = struct{}{}
		userIDs = append(userIDs, userID)
	}
	appendUnique(actorID)
	appendUnique(expense.PayByUserId)
	for _, allocation := range allocations {
		appendUnique(allocation.UserID)
	}
	return userIDs
}

func allocateEqualUnits(totalUnits int64, shares []int64) {
	count := int64(len(shares))
	base, remainder := totalUnits/count, totalUnits%count
	for index := range shares {
		shares[index] = base
		if int64(index) < remainder {
			shares[index]++
		}
	}
}

func allocatePercentageUnits(totalUnits int64, allocations []types.ExpenseAllocation, shares []int64) error {
	type fractionalShare struct {
		index     int
		remainder int64
		userID    string
	}
	var percentageTotal int64
	var allocatedUnits int64
	fractions := make([]fractionalShare, 0, len(allocations))
	for index, allocation := range allocations {
		if allocation.Amount != nil || allocation.PercentageBasisPoints == nil {
			return types.ErrInvalidMoney
		}
		basisPoints := int64(*allocation.PercentageBasisPoints)
		if basisPoints < 0 || basisPoints > 10_000 {
			return types.ErrInvalidMoney
		}
		percentageTotal += basisPoints
		numerator := totalUnits * basisPoints
		shares[index] = numerator / 10_000
		allocatedUnits += shares[index]
		fractions = append(fractions, fractionalShare{
			index: index, remainder: numerator % 10_000, userID: allocation.UserID.String(),
		})
	}
	if percentageTotal != 10_000 {
		return types.ErrInvalidMoney
	}
	slices.SortFunc(fractions, func(a, b fractionalShare) int {
		return cmp.Or(cmp.Compare(b.remainder, a.remainder), cmp.Compare(a.userID, b.userID))
	})
	for index := range totalUnits - allocatedUnits {
		shares[fractions[index].index]++
	}
	return nil
}

func validMoney(value decimal.Decimal, amountDigits int32) bool {
	return value.Abs().LessThanOrEqual(maximumMoneyValue) && value.Truncate(amountDigits).Equal(value)
}
