package tools

import (
	"errors"

	"expense-tracker/agent/internal/models"
)

type Code string

const (
	Validation      Code = "validation"
	Unauthenticated Code = "unauthenticated"
	Forbidden       Code = "forbidden"
	NotFound        Code = "not_found"
	Unavailable     Code = "unavailable"
	Internal        Code = "internal"
	Limit           Code = "limit_exceeded"
)

// Error contains only a stable public code, never transport/provider details.
type Error struct {
	Code Code
}

func (failure *Error) Error() string {
	return string(failure.Code)
}

func PublicCode(err error) Code {
	var failure *Error
	if errors.As(err, &failure) {
		switch failure.Code {
		case Validation, Unauthenticated, Forbidden, NotFound, Unavailable, Internal, Limit:
			return failure.Code
		default:
			return Internal
		}
	}
	if errors.Is(err, models.ErrBudget) {
		return Limit
	}
	return Unavailable
}
