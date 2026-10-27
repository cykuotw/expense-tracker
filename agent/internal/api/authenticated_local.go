package api

import (
	"fmt"
	"net/http"
	"strings"

	"expense-tracker/agent/internal/clients/worker"
	"expense-tracker/internal/localonly"
)

// NewLocal accepts access bearer tokens for explicit loopback testing. It is
// separate from the unauthenticated fixture and future browser-cookie handling.
func NewLocal(application Application) (http.Handler, error) {
	if err := localonly.CheckRuntime(); err != nil {
		return nil, err
	}
	if application == nil {
		return nil, fmt.Errorf("application required")
	}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /chat", func(writer http.ResponseWriter, request *http.Request) {
		writer.Header().Set("Cache-Control", "no-store")
		if !localonly.RequestAllowed(request) {
			http.Error(writer, "local requests only", http.StatusForbidden)
			return
		}
		for _, header := range []string{"Cookie", "X-Actor-ID", "X-User-ID", "X-Agent-Access-Token"} {
			if len(request.Header.Values(header)) != 0 {
				http.Error(writer, "unsupported credentials", http.StatusBadRequest)
				return
			}
		}
		authorization := request.Header.Values("Authorization")
		if len(authorization) != 1 {
			http.Error(writer, "access token required", http.StatusUnauthorized)
			return
		}
		scheme, access, found := strings.Cut(authorization[0], " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || access == "" || len(access) > 8192 || strings.ContainsAny(access, " \t\r\n") {
			http.Error(writer, "access token required", http.StatusUnauthorized)
			return
		}
		chat, valid := decodeChatRequest(writer, request)
		if !valid {
			return
		}
		if _, supported := writer.(http.Flusher); !supported {
			http.Error(writer, "streaming unsupported", http.StatusInternalServerError)
			return
		}
		stream := newChatStream(writer, chat.ConversationID)
		err := application.Run(worker.WithAccessToken(request.Context(), access), chat, stream.emit)
		if err != nil && request.Context().Err() == nil {
			stream.fail(err)
		}
	})
	return mux, nil
}
