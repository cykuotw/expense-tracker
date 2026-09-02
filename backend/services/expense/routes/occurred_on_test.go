package expense

import (
	"testing"
	"time"

	"expense-tracker/backend/types"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateOccurredOn(t *testing.T) {
	for _, value := range []string{"2026-01-01", "2024-02-29", "2000-12-31"} {
		t.Run(value, func(t *testing.T) {
			actual, err := validateOccurredOn(value)
			require.NoError(t, err)
			assert.Equal(t, value, actual)
		})
	}

	for _, value := range []string{"", "2026-1-01", "2026-02-29", "2026-13-01", "0000-01-01", "2026-01-01T00:00:00Z"} {
		t.Run("reject_"+value, func(t *testing.T) {
			_, err := validateOccurredOn(value)
			assert.ErrorIs(t, err, types.ErrInvalidOccurredOn)
		})
	}
}

func TestResolveCreateOccurredOnUsesUTCDayForLegacyOmission(t *testing.T) {
	now := time.Date(2026, time.January, 1, 23, 30, 0, 0, time.FixedZone("EST", -5*60*60))

	actual, err := resolveCreateOccurredOn(nil, now)

	require.NoError(t, err)
	assert.Equal(t, "2026-01-02", actual)
}

func TestResolveCreateOccurredOnRejectsExplicitEmptyValue(t *testing.T) {
	value := ""

	_, err := resolveCreateOccurredOn(&value, time.Now())

	assert.ErrorIs(t, err, types.ErrInvalidOccurredOn)
}
