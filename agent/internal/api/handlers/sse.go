package handlers

import (
	"fmt"
	"net/http"
	"time"

	"expense-tracker/agent/internal/models"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"github.com/google/uuid"
)

const maxEventBytes = 32 << 10

// chatStream owns the AG-UI lifecycle for a single HTTP response.
type chatStream struct {
	writer         http.ResponseWriter
	conversationID string
	runID          string
	messageID      string
	started        bool
	messageStarted bool
	terminal       bool
}

func newChatStream(writer http.ResponseWriter, conversationID string) *chatStream {
	return &chatStream{
		writer:         writer,
		conversationID: conversationID,
		messageID:      uuid.New().String(),
	}
}

func (stream *chatStream) emit(output models.Output) error {
	_ = http.NewResponseController(stream.writer).SetWriteDeadline(time.Now().Add(3 * time.Second))
	if output.Kind == "started" {
		stream.writer.Header().Set("Content-Type", "text/event-stream")
	}
	if output.MessageID != "" {
		stream.messageID = output.MessageID
	}
	if output.EventID != "" {
		if _, err := fmt.Fprintf(stream.writer, "id: %s\n", output.EventID); err != nil {
			return err
		}
	}
	switch output.Kind {
	case "started":
		stream.writer.Header().Set("Content-Type", "text/event-stream")
		stream.runID = output.RunID
		stream.started = true
		return writeEvent(stream.writer, events.NewRunStartedEvent(
			stream.conversationID, stream.runID,
		))

	case "text":
		if !stream.messageStarted {
			stream.messageStarted = true
			start := events.NewTextMessageStartEvent(
				stream.messageID, events.WithRole("assistant"),
			)
			if err := writeEvent(stream.writer, start); err != nil {
				return err
			}
		}
		return writeEvent(stream.writer, events.NewTextMessageContentEvent(
			stream.messageID, output.Delta,
		))

	case "finished":
		stream.terminal = true
		if err := stream.finishMessage(); err != nil {
			return err
		}
		return writeEvent(stream.writer, events.NewRunFinishedEvent(
			stream.conversationID, stream.runID,
		))
	case "error":
		stream.failCode(output.ErrorCode)
	}
	return nil
}

func (stream *chatStream) finishMessage() error {
	if !stream.messageStarted {
		return nil
	}
	if err := writeEvent(stream.writer, events.NewTextMessageEndEvent(stream.messageID)); err != nil {
		return err
	}
	stream.messageStarted = false
	return nil
}

func (stream *chatStream) fail(err error) {
	if stream.terminal {
		return
	}
	if !stream.started {
		status := http.StatusServiceUnavailable
		switch models.PublicCode(err) {
		case models.Unauthenticated:
			status = http.StatusUnauthorized
		case models.Forbidden:
			status = http.StatusForbidden
		case models.Validation:
			status = http.StatusBadRequest
		case models.NotFound:
			status = http.StatusNotFound
		case models.Limit:
			status = http.StatusTooManyRequests
		}
		http.Error(stream.writer, "run rejected", status)
		return
	}
	stream.failCode(string(models.PublicCode(err)))
}

func (stream *chatStream) failCode(code string) {
	if stream.terminal {
		return
	}
	stream.terminal = true
	if !stream.started {
		http.Error(stream.writer, "run rejected", http.StatusServiceUnavailable)
		return
	}
	if stream.messageStarted {
		_ = writeEvent(stream.writer, events.NewTextMessageEndEvent(stream.messageID))
	}
	failure := events.NewRunErrorEvent(
		"The agent could not complete this request.",
		events.WithErrorCode(code),
		events.WithRunID(stream.runID),
	)
	_ = writeEvent(stream.writer, failure)
}

func writeEvent(writer http.ResponseWriter, event events.Event) error {
	_ = http.NewResponseController(writer).SetWriteDeadline(time.Now().Add(3 * time.Second))
	if err := event.Validate(); err != nil {
		return err
	}
	payload, err := event.ToJSON()
	if err != nil {
		return err
	}
	if len(payload) > maxEventBytes {
		return fmt.Errorf("event exceeds payload limit")
	}
	if _, err := fmt.Fprintf(writer, "data: %s\n\n", payload); err != nil {
		return err
	}
	return http.NewResponseController(writer).Flush()
}
