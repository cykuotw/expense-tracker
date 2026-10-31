package handlers

import (
	"encoding/json/v2"
	"io"
	"mime"
	"net/http"

	"expense-tracker/agent/internal/models"
)

const maxChatBodyBytes = 16 << 10

func (handler *Handler) HandleChat(writer http.ResponseWriter, request *http.Request) {
	chat, valid := decodeChatRequest(writer, request)
	if !valid {
		return
	}
	if _, supported := writer.(http.Flusher); !supported {
		http.Error(writer, "streaming unsupported", http.StatusInternalServerError)
		return
	}
	stream := newChatStream(writer, chat.ConversationID)
	err := handler.application.Run(request.Context(), chat, stream.emit)
	if err != nil && request.Context().Err() == nil {
		stream.fail(err)
	}
}

func decodeChatRequest(
	writer http.ResponseWriter,
	request *http.Request,
) (models.ChatRequest, bool) {
	var chatRequest models.ChatRequest

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
