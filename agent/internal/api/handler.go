package api

import (
	"context"
	"encoding/json/v2"
	"io"
	"mime"
	"net/http"

	agentruntime "expense-tracker/agent/internal/runtime"
)

const maxChatBodyBytes = 16 << 10

type Application interface {
	Run(ctx context.Context, request agentruntime.Request, emit func(agentruntime.Output) error) error
}

type localFixtureHandler struct {
	application Application
}

// NewLocalFixture is intentionally not a production/authenticated handler.
// It rejects remote peers, foreign Host/Origin values, and identity headers.
func NewLocalFixture(application Application) http.Handler {
	handler := localFixtureHandler{application: application}
	mux := http.NewServeMux()
	mux.HandleFunc("POST /chat", handler.serveChat)
	return mux
}

func (handler localFixtureHandler) serveChat(writer http.ResponseWriter, request *http.Request) {
	writer.Header().Set("Cache-Control", "no-store")

	if !isLocalFixtureRequest(request) {
		http.Error(writer, "local fixture requests only", http.StatusForbidden)
		return
	}
	origin := request.Header.Get("Origin")
	if origin != "" && origin != "http://"+request.Host {
		http.Error(writer, "origin rejected", http.StatusForbidden)
		return
	}

	chatRequest, valid := decodeChatRequest(writer, request)
	if !valid {
		return
	}
	if _, supported := writer.(http.Flusher); !supported {
		http.Error(writer, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	stream := newChatStream(writer, chatRequest.ConversationID)
	err := handler.application.Run(request.Context(), chatRequest, stream.emit)
	if err != nil && request.Context().Err() == nil {
		stream.fail(err)
	}
}

func decodeChatRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (agentruntime.Request, bool) {
	var chatRequest agentruntime.Request

	mediaType, _, err := mime.ParseMediaType(request.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		http.Error(writer, "application/json required", http.StatusUnsupportedMediaType)
		return chatRequest, false
	}

	body := http.MaxBytesReader(writer, request.Body, maxChatBodyBytes)
	payload, err := io.ReadAll(body)
	if err != nil {
		http.Error(writer, "request too large", http.StatusRequestEntityTooLarge)
		return chatRequest, false
	}

	err = json.Unmarshal(payload, &chatRequest, json.RejectUnknownMembers(true))
	if err != nil {
		http.Error(writer, "invalid request", http.StatusBadRequest)
		return chatRequest, false
	}
	if err := chatRequest.Validate(); err != nil {
		http.Error(writer, "invalid request", http.StatusBadRequest)
		return chatRequest, false
	}
	return chatRequest, true
}
