package observability

import (
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
)

const (
	EventUnexpectedHTTPError   = "unexpected_http_error"
	EventHandlerReturnedError  = "handler_returned_error"
	defaultErrorCategory       = "unclassified_error"
	defaultDiagnosticMessage   = "unexpected internal failure"
	maxErrorCategoryLength     = 64
	maxDiagnosticMessageLength = 160

	httpErrorHandledKey = "observability_http_error_handled"
	httpErrorLoggedKey  = "observability_http_error_logged"
)

type diagnosticError struct {
	cause    error
	category string
	message  string
}

func (e *diagnosticError) Error() string {
	return e.cause.Error()
}

func (e *diagnosticError) Unwrap() error {
	return e.cause
}

// NewDiagnosticError wraps an internal error with a deliberately safe
// developer-authored classification. Category and message must be static and
// must never contain request, user, credential, or external-provider data.
func NewDiagnosticError(category string, message string, cause error) error {
	if cause == nil {
		return nil
	}
	if !validErrorCategory(category) {
		category = defaultErrorCategory
	}
	if !validDiagnosticMessage(message) {
		message = defaultDiagnosticMessage
	}
	return &diagnosticError{
		cause:    cause,
		category: category,
		message:  message,
	}
}

// RecordHTTPError marks an error response as handled and emits unexpected
// server failures at most once for the current request.
func RecordHTTPError(c *gin.Context, status int, code string, err error) {
	if err == nil {
		return
	}
	c.Set(httpErrorHandledKey, true)
	if status < http.StatusInternalServerError || errorWasLogged(c) {
		return
	}
	c.Set(httpErrorLoggedKey, true)

	category, message, errorType := classifyError(err)
	Logger(c).Error(EventUnexpectedHTTPError,
		slog.Bool("alertable", true),
		slog.String("method", requestMethod(c)),
		slog.String("route", MatchedRoute(c)),
		slog.Int("status", status),
		slog.String("error_code", code),
		slog.String("error_category", category),
		slog.String("error_type", errorType),
		slog.String("diagnostic_message", message),
	)
}

// RecordUnhandledHandlerError records a safe, non-alertable warning when a
// legacy error-returning handler did not write an error response itself.
func RecordUnhandledHandlerError(c *gin.Context, err error) {
	if err == nil || httpErrorWasHandled(c) {
		return
	}
	c.Set(httpErrorHandledKey, true)
	category, _, errorType := classifyError(err)
	Logger(c).Warn(EventHandlerReturnedError,
		slog.Bool("alertable", false),
		slog.String("method", requestMethod(c)),
		slog.String("route", MatchedRoute(c)),
		slog.Int("status", c.Writer.Status()),
		slog.String("error_category", category),
		slog.String("error_type", errorType),
	)
}

func httpErrorWasHandled(c *gin.Context) bool {
	value, exists := c.Get(httpErrorHandledKey)
	handled, _ := value.(bool)
	return exists && handled
}

func errorWasLogged(c *gin.Context) bool {
	value, exists := c.Get(httpErrorLoggedKey)
	logged, _ := value.(bool)
	return exists && logged
}

func classifyError(err error) (string, string, string) {
	category := defaultErrorCategory
	message := defaultDiagnosticMessage
	errorType := fmt.Sprintf("%T", err)

	var diagnostic *diagnosticError
	if errors.As(err, &diagnostic) {
		category = diagnostic.category
		message = diagnostic.message
		errorType = fmt.Sprintf("%T", diagnostic.cause)
	}
	return category, message, errorType
}

func validErrorCategory(value string) bool {
	if value == "" || len(value) > maxErrorCategoryLength {
		return false
	}
	for _, char := range value {
		if char < 'a' || char > 'z' {
			if char < '0' || char > '9' {
				if char != '_' {
					return false
				}
			}
		}
	}
	return true
}

func validDiagnosticMessage(value string) bool {
	if value == "" || len(value) > maxDiagnosticMessageLength {
		return false
	}
	for _, char := range value {
		if char < ' ' || char > '~' {
			return false
		}
	}
	return true
}
