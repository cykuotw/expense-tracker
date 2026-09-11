package observability

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"runtime/debug"
	"syscall"

	"github.com/gin-gonic/gin"
)

const (
	EventPanicRecovered         = "panic_recovered"
	EventClientConnectionClosed = "client_connection_closed"
	maxPanicStackBytes          = 64 * 1024
)

// Recovery catches handler panics without dumping the request or recovered
// value. Broken client connections are warnings and are never alertable.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}

			if isClosedClientConnection(recovered) {
				Logger(c).Warn(EventClientConnectionClosed,
					slog.Bool("alertable", false),
					slog.String("method", requestMethod(c)),
					slog.String("route", MatchedRoute(c)),
					slog.String("error_category", "client_connection_closed"),
				)
				c.Abort()
				return
			}

			c.Set(httpErrorHandledKey, true)
			c.Set(httpErrorLoggedKey, true)
			stack := debug.Stack()
			stackTruncated := len(stack) > maxPanicStackBytes
			stack = stack[:min(len(stack), maxPanicStackBytes)]
			Logger(c).Error(EventPanicRecovered,
				slog.Bool("alertable", true),
				slog.String("method", requestMethod(c)),
				slog.String("route", MatchedRoute(c)),
				slog.Int("status", http.StatusInternalServerError),
				slog.String("error_code", "internal_error"),
				slog.String("error_category", "panic"),
				slog.String("error_type", fmt.Sprintf("%T", recovered)),
				slog.String("diagnostic_message", "panic recovered"),
				slog.String("stack", string(stack)),
				slog.Bool("stack_truncated", stackTruncated),
			)
			c.AbortWithStatus(http.StatusInternalServerError)
		}()

		c.Next()
	}
}

func isClosedClientConnection(recovered any) bool {
	err, ok := recovered.(error)
	if !ok {
		return false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, net.ErrClosed) {
		return true
	}

	var networkError *net.OpError
	if !errors.As(err, &networkError) {
		return false
	}
	var syscallError *os.SyscallError
	if !errors.As(networkError, &syscallError) {
		return false
	}
	return errors.Is(syscallError.Err, syscall.EPIPE) || errors.Is(syscallError.Err, syscall.ECONNRESET)
}
