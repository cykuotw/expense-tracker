package types

import "errors"

var (
	// general
	ErrEmptyRequestBody = errors.New("missing request body")
	ErrPermissionDenied = errors.New("permission denied")

	// users
	ErrUserNotExist     = errors.New("invalid username/email/password")
	ErrPasswordNotMatch = errors.New("invalid username/email/password")

	// jwt
	ErrInvalidToken = errors.New("invalid token")

	// group
	ErrGroupNotExist    = errors.New("invalid group")
	ErrInvalidAction    = errors.New("invalid actions")
	ErrUserNotPermitted = errors.New("user has no permission")

	// expense
	ErrExpenseNotExist     = errors.New("expense not exist")
	ErrNoRemainingExpenses = errors.New("no remaining expenses in the list")
	ErrProviderNotExist    = errors.New("provider not exist")
)
