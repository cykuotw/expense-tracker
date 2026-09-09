package expense

import (
	"cmp"
	"slices"

	"expense-tracker/backend/types"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

var maximumMoneyValue = decimal.NewFromInt(10_000_000).Sub(decimal.New(1, -3))
var maximumItemQuantity = decimal.NewFromInt(100_000_000).Sub(decimal.New(1, -2))

func validateExpenseMoney(store types.ExpenseStore, expense types.Expense, items []types.Item, ledgers []types.Ledger, actorID uuid.UUID) error {
	amountDigits, err := store.GetCurrencyAmountDigits(expense.Currency)
	if err != nil {
		return err
	}

	if expense.Total.LessThanOrEqual(decimal.Zero) || !validMoney(expense.Total, amountDigits) ||
		!validMoney(expense.SubTotal, amountDigits) || !validMoney(expense.TaxFeeTip, amountDigits) {
		return types.ErrInvalidMoney
	}

	itemSubtotal := decimal.Zero
	for _, item := range items {
		if item.Amount.LessThanOrEqual(decimal.Zero) ||
			item.Amount.Abs().GreaterThan(maximumItemQuantity) ||
			!item.Amount.Truncate(2).Equal(item.Amount) ||
			item.UnitPrice.IsNegative() ||
			!validMoney(item.UnitPrice, amountDigits) {
			return types.ErrInvalidMoney
		}
		itemSubtotal = itemSubtotal.Add(item.Amount.Mul(item.UnitPrice).Round(amountDigits))
	}
	if len(items) > 0 && !itemSubtotal.Equal(expense.SubTotal) {
		return types.ErrInvalidMoney
	}

	if len(ledgers) == 0 {
		return types.ErrInvalidMoney
	}

	seenBorrowers := make(map[string]struct{}, len(ledgers))
	sum := decimal.Zero
	for _, ledger := range ledgers {
		if ledger.LenderUserID != expense.PayByUserId || ledger.Share.IsNegative() || !validMoney(ledger.Share, amountDigits) {
			return types.ErrInvalidMoney
		}
		borrowerID := ledger.BorrowerUesrID.String()
		if _, exists := seenBorrowers[borrowerID]; exists {
			return types.ErrInvalidMoney
		}
		seenBorrowers[borrowerID] = struct{}{}
		sum = sum.Add(ledger.Share)
	}
	if !sum.Equal(expense.Total) {
		return types.ErrInvalidMoney
	}
	return validateAutomaticSplit(expense, ledgers, amountDigits, actorID)
}

func validMoney(value decimal.Decimal, amountDigits int32) bool {
	return value.Abs().LessThanOrEqual(maximumMoneyValue) && value.Truncate(amountDigits).Equal(value)
}

func validateAutomaticSplit(expense types.Expense, ledgers []types.Ledger, amountDigits int32, actorID uuid.UUID) error {
	switch expense.SplitRule {
	case "Unequally", "":
		return nil
	case "Equally":
		return validateEqualShares(expense.Total, ledgers, amountDigits)
	case "You-Half", "Other-Half":
		if !validTwoPersonRuleParticipants(expense, ledgers, actorID) ||
			(expense.SplitRule == "You-Half") != (expense.PayByUserId == actorID) {
			return types.ErrInvalidMoney
		}
		return validateEqualShares(expense.Total, ledgers, amountDigits)
	case "You-Full", "Other-Full":
		if !validTwoPersonRuleParticipants(expense, ledgers, actorID) ||
			(expense.SplitRule == "You-Full") != (expense.PayByUserId == actorID) {
			return types.ErrInvalidMoney
		}
		for _, ledger := range ledgers {
			expected := decimal.Zero
			if ledger.BorrowerUesrID == expense.PayByUserId {
				expected = expense.Total
			}
			if !ledger.Share.Equal(expected) {
				return types.ErrInvalidMoney
			}
		}
		return nil
	default:
		return types.ErrInvalidAction
	}
}

func validateEqualShares(total decimal.Decimal, ledgers []types.Ledger, amountDigits int32) error {
	ordered := slices.Clone(ledgers)
	slices.SortFunc(ordered, func(a, b types.Ledger) int {
		return cmp.Compare(a.BorrowerUesrID.String(), b.BorrowerUesrID.String())
	})
	totalUnits := total.Shift(amountDigits).IntPart()
	count := int64(len(ordered))
	base, remainder := totalUnits/count, totalUnits%count
	for index, ledger := range ordered {
		expected := base
		if int64(index) < remainder {
			expected++
		}
		if ledger.Share.Shift(amountDigits).IntPart() != expected {
			return types.ErrInvalidMoney
		}
	}
	return nil
}

func validTwoPersonRuleParticipants(expense types.Expense, ledgers []types.Ledger, actorID uuid.UUID) bool {
	if len(ledgers) != 2 {
		return false
	}

	hasActor := false
	hasPayer := false
	for _, ledger := range ledgers {
		hasActor = hasActor || ledger.BorrowerUesrID == actorID
		hasPayer = hasPayer || ledger.BorrowerUesrID == expense.PayByUserId
	}
	return hasActor && hasPayer
}

func expenseParticipantIDs(actorID uuid.UUID, expense types.Expense, ledgers []types.Ledger) []uuid.UUID {
	userIDs := make([]uuid.UUID, 0, 2+len(ledgers)*2)
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
	for _, ledger := range ledgers {
		appendUnique(ledger.LenderUserID)
		appendUnique(ledger.BorrowerUesrID)
	}
	return userIDs
}
