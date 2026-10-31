package middleware

import (
	"context"
	"net/http"
	"strings"

	"expense-tracker/agent/internal/credentials"
	"expense-tracker/internal/localonly"
)

func WithLocalAccess(next http.HandlerFunc) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		ctx, allowed := localAccess(writer, request)
		if !allowed {
			return
		}
		next(writer, request.WithContext(ctx))
	}
}

func localAccess(writer http.ResponseWriter, request *http.Request) (context.Context, bool) {
	writer.Header().Set("Cache-Control", "no-store")
	if !localonly.RequestAllowed(request) {
		http.Error(writer, "local requests only", 403)
		return nil, false
	}
	for _, header := range []string{"Cookie", "X-Actor-ID", "X-User-ID", "X-Agent-Access-Token"} {
		if len(request.Header.Values(header)) != 0 {
			http.Error(writer, "unsupported credentials", 400)
			return nil, false
		}
	}
	values := request.Header.Values("Authorization")
	if len(values) != 1 {
		http.Error(writer, "access token required", 401)
		return nil, false
	}
	scheme, access, found := strings.Cut(values[0], " ")
	if !found || !strings.EqualFold(scheme, "Bearer") || access == "" || len(access) > 8192 || strings.ContainsAny(access, " \t\r\n") {
		http.Error(writer, "access token required", 401)
		return nil, false
	}
	return credentials.WithAccessToken(request.Context(), access), true
}
