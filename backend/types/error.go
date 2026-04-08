package types

import "errors"

var (
	// general
	ErrEmptyRequestBody = errors.New("missing request body")
	ErrPermissionDenied = errors.New("permission denied")

	// users
	ErrUserNotExist               = errors.New("invalid username/email/password")
	ErrPasswordNotMatch           = errors.New("invalid username/email/password")
	ErrInvalidCSRFToken           = errors.New("invalid csrf token")
	ErrInvalidJWTToken            = errors.New("invalid jwt token")
	ErrGoogleClaimsUnavailable    = errors.New("google verified claims are unavailable")
	ErrMissingAuthorizationHeader = errors.New("missing authorization header")
	ErrInvalidAuthorizationHeader = errors.New("invalid authorization header")
	ErrMissingBearerToken         = errors.New("missing bearer token")
	ErrMissingGoogleSubject       = errors.New("missing google subject claim")
	ErrMissingGoogleEmail         = errors.New("google email claim is required")
	ErrInvalidGoogleIDToken       = errors.New("invalid google id token")
	ErrInvalidGoogleIssuer        = errors.New("invalid google token issuer")
	ErrGoogleEmailNotVerified     = errors.New("google email must be verified")
	ErrGoogleAccountConflict      = errors.New("google account conflicts with an existing user")

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

	// balance
	ErrBalanceNotExist = errors.New("balacne not exist")
)

type ServerErr struct {
	Error string `json:"error"`
}
