package models

import "errors"

var (
	ErrMissing  = errors.New("conversation not found")
	ErrConflict = errors.New("run conflict")
	ErrStorage  = errors.New("agent storage unavailable")
	ErrPayload  = errors.New("agent payload limit exceeded")
	ErrBudget   = errors.New("execution budget exceeded")
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
	if errors.Is(err, ErrMissing) {
		return NotFound
	}
	if errors.Is(err, ErrConflict) || errors.Is(err, ErrPayload) {
		return Limit
	}
	if failure, ok := errors.AsType[*Error](err); ok {
		switch failure.Code {
		case Validation, Unauthenticated, Forbidden, NotFound, Unavailable, Internal, Limit:
			return failure.Code
		default:
			return Internal
		}
	}
	if errors.Is(err, ErrBudget) {
		return Limit
	}
	return Unavailable
}
