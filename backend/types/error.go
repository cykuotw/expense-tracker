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
	ErrGoogleLinkEmailMismatch    = errors.New("use the Google account that matches this account email")
	ErrGoogleAlreadyConnected     = errors.New("a Google account is already connected")
	ErrGoogleLinkUnavailable      = errors.New("google linking is unavailable for this account")
	ErrInvitationRequired         = errors.New("a valid invitation is required to create an account")
	ErrInvitationInvalid          = errors.New("invitation is invalid")
	ErrInvitationExpired          = errors.New("invitation has expired")
	ErrInvitationUsed             = errors.New("invitation has already been used")
	ErrInvitationEmailMismatch    = errors.New("use the Google account that matches the invitation email")
	ErrAccountConflict            = errors.New("an account already exists for this identity")
	ErrAccountInactive            = errors.New("account is inactive")
	ErrCannotDeactivateSelf       = errors.New("administrators cannot deactivate their own account")
	ErrLastActiveAdmin            = errors.New("the last active administrator cannot be deactivated")
	ErrCannotChangeOwnRole        = errors.New("administrators cannot change their own role")
	ErrProtectedAdmin             = errors.New("the system owner cannot be managed")
	ErrInvalidUserRole            = errors.New("invalid user role")
	ErrInvalidProfile             = errors.New("first name and last name are required")
	ErrCurrentPasswordIncorrect   = errors.New("current password is incorrect")
	ErrPasswordChangeUnavailable  = errors.New("password changes are unavailable for this account")
	ErrPasswordUnchanged          = errors.New("new password must be different from the current password")
	ErrInvalidPasswordLength      = errors.New("password must be between 8 and 72 bytes")

	// jwt
	ErrInvalidToken = errors.New("invalid token")

	// group
	ErrGroupNotExist        = errors.New("invalid group")
	ErrInvalidAction        = errors.New("invalid actions")
	ErrUserNotPermitted     = errors.New("user has no permission")
	ErrProtectedGroupMember = errors.New("group creator and final member cannot be removed")

	// expense
	ErrExpenseNotExist            = errors.New("expense not exist")
	ErrItemNotExist               = errors.New("item not found for expense")
	ErrLedgerNotExist             = errors.New("ledger not found for expense")
	ErrNoRemainingExpenses        = errors.New("no remaining expenses in the list")
	ErrProviderNotExist           = errors.New("provider not exist")
	ErrGroupParticipantNotAllowed = errors.New("expense participants must belong to the group")
	ErrInvalidIdempotencyKey      = errors.New("a valid idempotency key is required")
	ErrIdempotencyKeyConflict     = errors.New("this submission key was already used for different expense details")

	// balance
	ErrBalanceNotExist = errors.New("balacne not exist")
)

type ServerErr struct {
	Error string `json:"error"`
}
