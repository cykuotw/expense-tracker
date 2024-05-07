package types

import "errors"

var (
	ErrEmptyRequestBody = errors.New("missing request body")

	// users
	ErrUserNotExist = errors.New("user does not exist")
)
