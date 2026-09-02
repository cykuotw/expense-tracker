package expense

import (
	"expense-tracker/backend/types"
	"time"
)

func resolveCreateOccurredOn(value *string, now time.Time) (string, error) {
	if value == nil {
		return now.UTC().Format(time.DateOnly), nil
	}
	return validateOccurredOn(*value)
}

func validateOccurredOn(value string) (string, error) {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil || parsed.Year() < 1 || parsed.Format(time.DateOnly) != value {
		return "", types.ErrInvalidOccurredOn
	}
	return value, nil
}
