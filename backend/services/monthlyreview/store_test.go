package monthlyreview

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestExpenseCursorRoundTrip(t *testing.T) {
	expected := expenseCursor{
		OccurredOn:     time.Date(2026, time.August, 20, 0, 0, 0, 0, time.UTC),
		ExpenseTimeUTC: time.Date(2026, time.August, 20, 14, 30, 0, 0, time.UTC),
		ID:             uuid.New(),
	}

	encoded, err := encodeExpenseCursor(expected)
	require.NoError(t, err)
	actual, err := decodeExpenseCursor(encoded)
	require.NoError(t, err)
	require.Equal(t, expected, actual)
}

func TestExpenseCursorRejectsInvalidValues(t *testing.T) {
	for _, value := range []string{"not-base64", "e30"} {
		_, err := decodeExpenseCursor(value)
		require.ErrorIs(t, err, ErrInvalidExpenseCursor)
	}
}

func TestExpenseCursorScopeAndSortCompatibility(t *testing.T) {
	groupID := uuid.New()
	month := time.Date(2026, time.August, 1, 0, 0, 0, 0, time.UTC)
	options := ExpenseListOptions{Category: "Food and Drink", Sort: "amount_asc"}
	cursor := expenseCursor{OccurredOn: month, ExpenseTimeUTC: month, ID: uuid.New(),
		Total: "10.25", Sort: options.Sort, Category: options.Category,
		Currency: "CAD", GroupID: groupID.String(), Month: "2026-08"}
	encoded, err := encodeExpenseCursor(cursor)
	require.NoError(t, err)
	decoded, err := decodeExpenseCursor(encoded)
	require.NoError(t, err)
	require.NoError(t, decoded.validateScope(groupID, month, "CAD", options))
	for _, changed := range []ExpenseListOptions{
		{Category: options.Category, Sort: "amount_desc"},
		{Category: "Transportation", Sort: options.Sort},
	} {
		require.ErrorIs(t, decoded.validateScope(groupID, month, "CAD", changed), ErrInvalidExpenseCursor)
	}
	require.ErrorIs(t, decoded.validateScope(uuid.New(), month, "CAD", options), ErrInvalidExpenseCursor)
	require.ErrorIs(t, decoded.validateScope(groupID, month.AddDate(0, 1, 0), "CAD", options), ErrInvalidExpenseCursor)
	require.ErrorIs(t, decoded.validateScope(groupID, month, "TWD", options), ErrInvalidExpenseCursor)
	decoded.Total = "invalid"
	require.ErrorIs(t, decoded.validateScope(groupID, month, "CAD", options), ErrInvalidExpenseCursor)
	legacy := expenseCursor{OccurredOn: month, ExpenseTimeUTC: month, ID: uuid.New()}
	require.NoError(t, legacy.validateScope(groupID, month, "CAD", ExpenseListOptions{Sort: "date_desc"}))
	require.ErrorIs(t, legacy.validateScope(groupID, month, "CAD", options), ErrInvalidExpenseCursor)
}
