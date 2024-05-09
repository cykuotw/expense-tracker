package types

import "errors"

var (
	ErrEmptyRequestBody = errors.New("missing request body")

	// users
	ErrUserNotExist     = errors.New("invalid username/email/password")
	ErrPasswordNotMatch = errors.New("invalid username/email/password")
)
