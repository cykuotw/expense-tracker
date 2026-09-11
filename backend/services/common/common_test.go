package common

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"expense-tracker/backend/internal/observability"
	"expense-tracker/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMakeDoesNotDuplicateHandledServerErrors(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	var output bytes.Buffer
	router := gin.New()
	router.Use(observability.RequestLogging(observability.RequestLoggingConfig{
		Logger:       observability.NewLogger("release", &output),
		NewRequestID: func() string { return "request-id" },
	}))
	router.GET("/failure", Make(func(c *gin.Context) error {
		err := observability.NewDiagnosticError(
			"database_query",
			"user lookup failed",
			errors.New("database-password-secret"),
		)
		utils.WriteError(c, http.StatusInternalServerError, err)
		return err
	}))

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/failure", nil))

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.JSONEq(t, `{"error":"internal server error","code":"internal_error"}`, recorder.Body.String())
	assert.Equal(t, 1, bytes.Count(output.Bytes(), []byte(observability.EventUnexpectedHTTPError)))
	assert.NotContains(t, output.String(), "database-password-secret")
}

func TestMakeSafelyWarnsForUnhandledReturnedError(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	var output bytes.Buffer
	router := gin.New()
	router.Use(observability.RequestLogging(observability.RequestLoggingConfig{
		Logger:       observability.NewLogger("release", &output),
		NewRequestID: func() string { return "request-id" },
	}))
	router.GET("/unhandled", Make(func(*gin.Context) error {
		return errors.New("returned-error-secret")
	}))

	router.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/unhandled", nil))

	assert.Equal(t, 1, bytes.Count(output.Bytes(), []byte(observability.EventHandlerReturnedError)))
	assert.NotContains(t, output.String(), "returned-error-secret")
	lines := bytes.Split(bytes.TrimSpace(output.Bytes()), []byte("\n"))
	require.Len(t, lines, 2)
	assert.Contains(t, string(lines[0]), `"alertable":false`)
}
