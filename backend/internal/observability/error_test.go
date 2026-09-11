package observability

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDiagnosticErrorPreservesCauseAndSafeClassification(t *testing.T) {
	cause := errors.New("database-password-secret")
	err := fmt.Errorf("query group: %w", NewDiagnosticError("database_query", "group lookup failed", cause))

	assert.ErrorIs(t, err, cause)
	category, message, errorType := classifyError(err)
	assert.Equal(t, "database_query", category)
	assert.Equal(t, "group lookup failed", message)
	assert.Equal(t, "*errors.errorString", errorType)
}

func TestNewDiagnosticErrorRejectsUnsafeClassification(t *testing.T) {
	err := NewDiagnosticError("INVALID\ncategory", "unsafe\tdiagnostic", errors.New("secret"))

	category, message, _ := classifyError(err)
	assert.Equal(t, defaultErrorCategory, category)
	assert.Equal(t, defaultDiagnosticMessage, message)
}

func TestRecordHTTPErrorLogsUnexpectedFailureExactlyOnce(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	var output bytes.Buffer
	router := gin.New()
	router.Use(RequestLogging(RequestLoggingConfig{
		Logger:       NewLogger(releaseMode, &output),
		NewRequestID: func() string { return "request-id" },
	}))
	router.GET("/groups/:id", func(c *gin.Context) {
		cause := errors.New("database-password-secret")
		err := NewDiagnosticError("database_query", "group lookup failed", cause)
		RecordHTTPError(c, http.StatusInternalServerError, "internal_error", err)
		RecordHTTPError(c, http.StatusInternalServerError, "internal_error", fmt.Errorf("wrapped: %w", err))
		c.Status(http.StatusInternalServerError)
	})

	request := httptest.NewRequest(http.MethodGet, "/groups/path-secret", nil)
	router.ServeHTTP(httptest.NewRecorder(), request)

	entries := decodeLogEntries(t, output.String())
	require.Len(t, entries, 2)
	assert.Equal(t, EventUnexpectedHTTPError, entries[0]["event"])
	assert.Equal(t, true, entries[0]["alertable"])
	assert.Equal(t, "/groups/:id", entries[0]["route"])
	assert.Equal(t, "internal_error", entries[0]["error_code"])
	assert.Equal(t, "database_query", entries[0]["error_category"])
	assert.Equal(t, "group lookup failed", entries[0]["diagnostic_message"])
	assert.Equal(t, EventRequestCompleted, entries[1]["event"])
	assert.NotContains(t, output.String(), "database-password-secret")
	assert.NotContains(t, output.String(), "path-secret")
}

func TestRecordHTTPErrorDoesNotLogExpectedClientError(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	var output bytes.Buffer
	router := gin.New()
	router.Use(RequestLogging(RequestLoggingConfig{
		Logger:       NewLogger(releaseMode, &output),
		NewRequestID: func() string { return "request-id" },
	}))
	router.GET("/expected", func(c *gin.Context) {
		RecordHTTPError(c, http.StatusForbidden, "forbidden", errors.New("authorization-secret"))
		c.Status(http.StatusForbidden)
	})

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/expected", nil))

	entries := decodeLogEntries(t, output.String())
	require.Len(t, entries, 1)
	assert.Equal(t, EventRequestCompleted, entries[0]["event"])
	assert.NotContains(t, output.String(), "authorization-secret")
}
