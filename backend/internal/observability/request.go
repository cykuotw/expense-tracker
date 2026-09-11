package observability

import (
	"context"
	"log/slog"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const (
	RequestIDHeader       = "X-Request-ID"
	EventRequestCompleted = "request_completed"
	EventSlowRequest      = "slow_request"

	requestIDKey     = "observability_request_id"
	requestLoggerKey = "observability_request_logger"
	unmatchedRoute   = "unmatched"

	maxTrustedRequestIDLength = 128
)

type runtimeRequestIDsKey struct{}

// RuntimeRequestIDs contains identifiers supplied by the trusted serverless
// runtime. These identifiers supplement, but never replace, the application ID.
type RuntimeRequestIDs struct {
	APIGateway string
	AWSLambda  string
}

// RequestLoggingConfig provides deterministic seams for middleware tests.
type RequestLoggingConfig struct {
	Logger               *slog.Logger
	NewRequestID         func() string
	Now                  func() time.Time
	SlowRequestThreshold time.Duration
}

// ContextWithRuntimeRequestIDs attaches trusted platform correlation metadata.
func ContextWithRuntimeRequestIDs(ctx context.Context, ids RuntimeRequestIDs) context.Context {
	return context.WithValue(ctx, runtimeRequestIDsKey{}, RuntimeRequestIDs{
		APIGateway: safeTrustedRequestID(ids.APIGateway),
		AWSLambda:  safeTrustedRequestID(ids.AWSLambda),
	})
}

// RequestLogging emits one completion event for each request and, when enabled,
// a separate event for requests at or above the slow-request threshold.
func RequestLogging(config RequestLoggingConfig) gin.HandlerFunc {
	logger := config.Logger
	if logger == nil {
		logger = slog.Default()
	}
	newRequestID := config.NewRequestID
	if newRequestID == nil {
		newRequestID = uuid.NewString
	}
	now := config.Now
	if now == nil {
		now = time.Now
	}

	return func(c *gin.Context) {
		startedAt := now()
		requestID := newRequestID()
		requestLogger := logger.With(slog.String("request_id", requestID))

		if ids, ok := RuntimeRequestIDsFromContext(c.Request.Context()); ok {
			if ids.APIGateway != "" {
				requestLogger = requestLogger.With(slog.String("api_gateway_request_id", ids.APIGateway))
			}
			if ids.AWSLambda != "" {
				requestLogger = requestLogger.With(slog.String("aws_request_id", ids.AWSLambda))
			}
		}

		c.Set(requestIDKey, requestID)
		c.Set(requestLoggerKey, requestLogger)
		c.Header(RequestIDHeader, requestID)
		c.Next()

		latency := max(time.Duration(0), now().Sub(startedAt))
		route := MatchedRoute(c)

		requestLogger.Info(EventRequestCompleted,
			slog.String("method", c.Request.Method),
			slog.String("route", route),
			slog.Int("status", c.Writer.Status()),
			slog.Int64("latency_ms", latency.Milliseconds()),
		)

		if config.SlowRequestThreshold > 0 && latency >= config.SlowRequestThreshold {
			requestLogger.Warn(EventSlowRequest,
				slog.String("route", route),
				slog.Int64("latency_ms", latency.Milliseconds()),
				slog.Int64("threshold_ms", config.SlowRequestThreshold.Milliseconds()),
			)
		}
	}
}

// RequestID returns the application-generated correlation ID for a Gin request.
func RequestID(c *gin.Context) string {
	requestID, _ := c.Get(requestIDKey)
	value, _ := requestID.(string)
	return value
}

// Logger returns the request-scoped logger, or the default logger outside the
// request middleware boundary.
func Logger(c *gin.Context) *slog.Logger {
	logger, _ := c.Get(requestLoggerKey)
	if value, ok := logger.(*slog.Logger); ok {
		return value
	}
	return slog.Default()
}

// MatchedRoute returns only the registered route template. Raw request paths
// and targets are intentionally never used as a fallback.
func MatchedRoute(c *gin.Context) string {
	if c == nil {
		return unmatchedRoute
	}
	route := c.FullPath()
	if route == "" {
		return unmatchedRoute
	}
	return route
}

func requestMethod(c *gin.Context) string {
	if c == nil || c.Request == nil || c.Request.Method == "" {
		return "unknown"
	}
	return c.Request.Method
}

// RuntimeRequestIDsFromContext returns trusted platform correlation metadata.
func RuntimeRequestIDsFromContext(ctx context.Context) (RuntimeRequestIDs, bool) {
	ids, ok := ctx.Value(runtimeRequestIDsKey{}).(RuntimeRequestIDs)
	return ids, ok
}

func safeTrustedRequestID(value string) string {
	if value == "" || len(value) > maxTrustedRequestIDLength {
		return ""
	}
	for _, char := range value {
		if !isTrustedRequestIDCharacter(char) {
			return ""
		}
	}
	return value
}

func isTrustedRequestIDCharacter(char rune) bool {
	return char >= 'a' && char <= 'z' ||
		char >= 'A' && char <= 'Z' ||
		char >= '0' && char <= '9' ||
		char == '-' || char == '_' || char == '.' || char == ':' ||
		char == '/' || char == '+' || char == '='
}
