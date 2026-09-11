package observability

import (
	"bytes"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"syscall"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecoveryLogsPanicWithoutRequestOrRecoveredValue(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	var output bytes.Buffer
	router := gin.New()
	router.Use(RequestLogging(RequestLoggingConfig{
		Logger:       NewLogger(releaseMode, &output),
		NewRequestID: func() string { return "request-id" },
	}))
	router.Use(Recovery())
	router.POST("/panic/:id", func(*gin.Context) {
		panic("panic-value-secret")
	})

	request := httptest.NewRequest(
		http.MethodPost,
		"/panic/path-secret?query=query-secret",
		strings.NewReader(`{"body":"body-secret"}`),
	)
	request.Header.Set("Authorization", "Bearer authorization-secret")
	request.Header.Set("Cookie", "refresh=refresh-secret; csrf=csrf-secret")
	request.Header.Set("X-Invitation-Token", "invitation-secret")
	request.Header.Set("X-Google-Subject", "google-subject-secret")
	request.Header.Set("X-User-Email", "user-email-secret@example.com")
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, http.StatusInternalServerError, recorder.Code)
	assert.Empty(t, recorder.Body.String())
	entries := decodeLogEntries(t, output.String())
	require.Len(t, entries, 2)
	assert.Equal(t, EventPanicRecovered, entries[0]["event"])
	assert.Equal(t, true, entries[0]["alertable"])
	assert.Equal(t, "/panic/:id", entries[0]["route"])
	assert.Equal(t, "internal_error", entries[0]["error_code"])
	assert.Equal(t, "panic", entries[0]["error_category"])
	assert.Equal(t, "string", entries[0]["error_type"])
	assert.Contains(t, entries[0]["stack"], "TestRecoveryLogsPanicWithoutRequestOrRecoveredValue")
	assert.Equal(t, EventRequestCompleted, entries[1]["event"])

	for _, secret := range []string{
		"panic-value-secret", "path-secret", "query-secret", "body-secret",
		"authorization-secret", "refresh-secret", "csrf-secret", "invitation-secret",
		"google-subject-secret", "user-email-secret@example.com",
	} {
		assert.NotContains(t, output.String(), secret)
	}
}

func TestRecoveryTreatsBrokenPipeAsNonAlertable(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	var output bytes.Buffer
	router := gin.New()
	router.Use(RequestLogging(RequestLoggingConfig{
		Logger:       NewLogger(releaseMode, &output),
		NewRequestID: func() string { return "request-id" },
	}))
	router.Use(Recovery())
	router.GET("/broken/:id", func(*gin.Context) {
		panic(&net.OpError{
			Op:  "broken-operation-secret",
			Net: "tcp",
			Err: &os.SyscallError{Syscall: "write-secret", Err: syscall.EPIPE},
		})
	})

	request := httptest.NewRequest(http.MethodGet, "/broken/path-secret?query-secret", nil)
	router.ServeHTTP(httptest.NewRecorder(), request)

	entries := decodeLogEntries(t, output.String())
	require.Len(t, entries, 2)
	assert.Equal(t, EventClientConnectionClosed, entries[0]["event"])
	assert.Equal(t, false, entries[0]["alertable"])
	assert.Equal(t, EventRequestCompleted, entries[1]["event"])
	assert.NotContains(t, output.String(), EventPanicRecovered)
	for _, secret := range []string{"broken-operation-secret", "write-secret", "path-secret", "query-secret"} {
		assert.NotContains(t, output.String(), secret)
	}
}
