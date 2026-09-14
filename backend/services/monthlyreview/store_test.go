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
