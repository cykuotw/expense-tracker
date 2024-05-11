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
	ErrGroupNotExist = errors.New("invalid group")
	ErrInvalidAction = errors.New("invalid actions")
)
