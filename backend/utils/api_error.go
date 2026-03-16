package utils

import (
	"encoding/json"
	"errors"
	"expense-tracker/backend/types"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

type APIErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code"`
	Details any    `json:"details,omitempty"`
}

type ValidationErrorDetail struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type detailedError struct {
	message string
	code    string
	details any
}

func (e *detailedError) Error() string {
	return e.message
}

func NewDetailedError(message string, code string, details any) error {
	return &detailedError{
		message: message,
		code:    code,
		details: details,
	}
}

func NewValidationError(errs validator.ValidationErrors) error {
	details := make([]ValidationErrorDetail, 0, len(errs))
	for _, err := range errs {
		details = append(details, ValidationErrorDetail{
			Field:   normalizeFieldName(err.Field()),
			Code:    err.Tag(),
			Message: validationMessage(err),
		})
	}

	return NewDetailedError("invalid payload", "invalid_payload", details)
}

func AbortWithError(c *gin.Context, status int, err error) {
	WriteError(c, status, err)
	c.Abort()
}

func normalizeError(status int, err error) APIErrorResponse {
	if err == nil {
		return APIErrorResponse{}
	}

	var detailed *detailedError
	if errors.As(err, &detailed) {
		return APIErrorResponse{
			Error:   detailed.message,
			Code:    detailed.code,
			Details: detailed.details,
		}
	}

	var syntaxErr *json.SyntaxError
	if errors.As(err, &syntaxErr) {
		return APIErrorResponse{
			Error: "invalid json payload",
			Code:  "invalid_json",
			Details: map[string]any{
				"offset": syntaxErr.Offset,
			},
		}
	}

	var typeErr *json.UnmarshalTypeError
	if errors.As(err, &typeErr) {
		details := map[string]any{
			"code":     "invalid_type",
			"expected": typeErr.Type.String(),
		}
		if typeErr.Field != "" {
			details["field"] = typeErr.Field
		}

		return APIErrorResponse{
			Error:   "invalid json payload",
			Code:    "invalid_json",
			Details: details,
		}
	}

	if errors.Is(err, types.ErrEmptyRequestBody) {
		return APIErrorResponse{
			Error: err.Error(),
			Code:  "empty_request_body",
		}
	}
	if errors.Is(err, types.ErrPermissionDenied) {
		return APIErrorResponse{Error: err.Error(), Code: "permission_denied"}
	}
	if errors.Is(err, types.ErrUserNotExist) || errors.Is(err, types.ErrPasswordNotMatch) {
		return APIErrorResponse{Error: err.Error(), Code: "invalid_credentials"}
	}
	if errors.Is(err, types.ErrInvalidCSRFToken) {
		return APIErrorResponse{Error: err.Error(), Code: "invalid_csrf_token"}
	}
	if errors.Is(err, types.ErrInvalidJWTToken) {
		return APIErrorResponse{Error: err.Error(), Code: "invalid_jwt_token"}
	}
	if errors.Is(err, types.ErrInvalidToken) {
		return APIErrorResponse{Error: err.Error(), Code: "invalid_token"}
	}
	if errors.Is(err, types.ErrGroupNotExist) {
		return APIErrorResponse{Error: err.Error(), Code: "invalid_group"}
	}
	if errors.Is(err, types.ErrInvalidAction) {
		return APIErrorResponse{Error: err.Error(), Code: "invalid_action"}
	}
	if errors.Is(err, types.ErrUserNotPermitted) {
		return APIErrorResponse{Error: err.Error(), Code: "user_not_permitted"}
	}
	if errors.Is(err, types.ErrExpenseNotExist) {
		return APIErrorResponse{Error: err.Error(), Code: "expense_not_exist"}
	}
	if errors.Is(err, types.ErrNoRemainingExpenses) {
		return APIErrorResponse{Error: err.Error(), Code: "no_remaining_expenses"}
	}
	if errors.Is(err, types.ErrProviderNotExist) {
		return APIErrorResponse{Error: err.Error(), Code: "provider_not_exist"}
	}
	if errors.Is(err, types.ErrBalanceNotExist) {
		return APIErrorResponse{Error: err.Error(), Code: "balance_not_exist"}
	}

	if status >= http.StatusInternalServerError {
		return APIErrorResponse{
			Error: "internal server error",
			Code:  "internal_error",
		}
	}

	return APIErrorResponse{
		Error: err.Error(),
		Code:  defaultCodeForStatus(status),
	}
}

func defaultCodeForStatus(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusConflict:
		return "conflict"
	default:
		return "request_error"
	}
}

func validationMessage(err validator.FieldError) string {
	switch err.Tag() {
	case "required":
		return "is required"
	case "email":
		return "must be a valid email"
	case "min":
		return fmt.Sprintf("must be at least %s characters", err.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", err.Param())
	default:
		return fmt.Sprintf("failed validation: %s", err.Tag())
	}
}

func normalizeFieldName(field string) string {
	if field == "" {
		return ""
	}
	runes := []rune(field)
	runes[0] = unicode.ToLower(runes[0])
	return strings.TrimSpace(string(runes))
}
