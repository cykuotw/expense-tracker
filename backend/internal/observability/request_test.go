package observability

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRequestLoggingUsesSafeRouteAndApplicationRequestID(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	var output bytes.Buffer
	baseTime := time.Date(2026, time.September, 11, 12, 0, 0, 0, time.UTC)
	times := []time.Time{baseTime, baseTime.Add(1500 * time.Millisecond)}
	nowIndex := 0

	router := gin.New()
	router.Use(RequestLogging(RequestLoggingConfig{
		Logger:       NewLogger(releaseMode, &output),
		NewRequestID: func() string { return "application-request-id" },
		Now: func() time.Time {
			value := times[nowIndex]
			nowIndex++
			return value
		},
		SlowRequestThreshold: time.Second,
	}))
	router.POST("/groups/:id", func(c *gin.Context) {
		assert.Equal(t, "application-request-id", RequestID(c))
		assert.NotNil(t, Logger(c))
		c.Status(http.StatusNoContent)
	})

	request := httptest.NewRequest(
		http.MethodPost,
		"/groups/path-secret?query=query-secret",
		strings.NewReader(`{"body":"body-secret"}`),
	)
	request.Header.Set(RequestIDHeader, "caller-request-secret")
	request.Header.Set("Authorization", "Bearer authorization-secret")
	request.Header.Set("Cookie", "session=cookie-secret; csrf=csrf-secret")
	request.Header.Set("X-Invitation-Token", "invitation-secret")
	request = request.WithContext(ContextWithRuntimeRequestIDs(request.Context(), RuntimeRequestIDs{
		APIGateway: "gateway-request-id",
		AWSLambda:  "lambda-request-id",
	}))
	recorder := httptest.NewRecorder()

	router.ServeHTTP(recorder, request)

	assert.Equal(t, "application-request-id", recorder.Header().Get(RequestIDHeader))
	entries := decodeLogEntries(t, output.String())
	require.Len(t, entries, 2)
	assert.Equal(t, "request_completed", entries[0]["event"])
	assert.Equal(t, "application-request-id", entries[0]["request_id"])
	assert.Equal(t, "gateway-request-id", entries[0]["api_gateway_request_id"])
	assert.Equal(t, "lambda-request-id", entries[0]["aws_request_id"])
	assert.Equal(t, http.MethodPost, entries[0]["method"])
	assert.Equal(t, "/groups/:id", entries[0]["route"])
	assert.Equal(t, float64(http.StatusNoContent), entries[0]["status"])
	assert.Equal(t, float64(1500), entries[0]["latency_ms"])
	assert.Equal(t, "slow_request", entries[1]["event"])
	assert.Equal(t, float64(1000), entries[1]["threshold_ms"])

	for _, secret := range []string{
		"path-secret", "query-secret", "body-secret", "caller-request-secret",
		"authorization-secret", "cookie-secret", "csrf-secret", "invitation-secret",
	} {
		assert.NotContains(t, output.String(), secret)
	}
}

func TestRequestLoggingDoesNotFallBackToUnmatchedRawPath(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	var output bytes.Buffer
	router := gin.New()
	router.Use(RequestLogging(RequestLoggingConfig{
		Logger:       NewLogger(releaseMode, &output),
		NewRequestID: func() string { return "request-id" },
	}))

	request := httptest.NewRequest(http.MethodGet, "/unmatched-path-secret?query-secret", nil)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	entries := decodeLogEntries(t, output.String())
	require.Len(t, entries, 1)
	assert.Equal(t, unmatchedRoute, entries[0]["route"])
	assert.NotContains(t, output.String(), "unmatched-path-secret")
	assert.NotContains(t, output.String(), "query-secret")
}

func TestContextWithRuntimeRequestIDsRejectsUnsafeValues(t *testing.T) {
	ctx := ContextWithRuntimeRequestIDs(context.Background(), RuntimeRequestIDs{
		APIGateway: "gateway-id\nforged-event",
		AWSLambda:  strings.Repeat("a", maxTrustedRequestIDLength+1),
	})

	ids, ok := RuntimeRequestIDsFromContext(ctx)
	require.True(t, ok)
	assert.Empty(t, ids.APIGateway)
	assert.Empty(t, ids.AWSLambda)
}

func TestRequestLoggingKeepsConcurrentRuntimeFieldsIsolated(t *testing.T) {
	gin.SetMode(gin.ReleaseMode)
	var output synchronizedBuffer
	var requestCount atomic.Int64
	router := gin.New()
	router.Use(RequestLogging(RequestLoggingConfig{
		Logger: NewLogger(releaseMode, &output),
		NewRequestID: func() string {
			return fmt.Sprintf("application-%d", requestCount.Add(1))
		},
	}))
	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})

	const concurrentRequests = 32
	var waitGroup sync.WaitGroup
	for index := range concurrentRequests {
		waitGroup.Go(func() {
			suffix := strconv.Itoa(index)
			request := httptest.NewRequest(http.MethodGet, "/health", nil)
			request = request.WithContext(ContextWithRuntimeRequestIDs(request.Context(), RuntimeRequestIDs{
				APIGateway: "gateway-" + suffix,
				AWSLambda:  "lambda-" + suffix,
			}))
			router.ServeHTTP(httptest.NewRecorder(), request)
		})
	}
	waitGroup.Wait()

	entries := decodeLogEntries(t, output.String())
	require.Len(t, entries, concurrentRequests)
	applicationIDs := make(map[string]struct{}, concurrentRequests)
	for _, entry := range entries {
		applicationIDs[entry["request_id"].(string)] = struct{}{}
		gatewayID := entry["api_gateway_request_id"].(string)
		suffix, ok := strings.CutPrefix(gatewayID, "gateway-")
		require.True(t, ok)
		assert.Equal(t, "lambda-"+suffix, entry["aws_request_id"])
	}
	assert.Len(t, applicationIDs, concurrentRequests)
}

type synchronizedBuffer struct {
	mutex  sync.Mutex
	buffer bytes.Buffer
}

func (b *synchronizedBuffer) Write(data []byte) (int, error) {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.Write(data)
}

func (b *synchronizedBuffer) String() string {
	b.mutex.Lock()
	defer b.mutex.Unlock()
	return b.buffer.String()
}
