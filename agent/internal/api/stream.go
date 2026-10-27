package api

import (
	"fmt"
	"net/http"

	agentruntime "expense-tracker/agent/internal/runtime"
	"expense-tracker/agent/internal/tools"
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
}

func newChatStream(writer http.ResponseWriter, conversationID string) *chatStream {
	return &chatStream{
		writer:         writer,
		conversationID: conversationID,
		messageID:      uuid.New().String(),
	}
}

func (stream *chatStream) emit(output agentruntime.Output) error {
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
		if err := stream.finishMessage(); err != nil {
			return err
		}
		return writeEvent(stream.writer, events.NewRunFinishedEvent(
			stream.conversationID, stream.runID,
		))
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
	if !stream.started {
		status := http.StatusServiceUnavailable
		switch tools.PublicCode(err) {
		case tools.Unauthenticated:
			status = http.StatusUnauthorized
		case tools.Forbidden:
			status = http.StatusForbidden
		case tools.Validation:
			status = http.StatusBadRequest
		case tools.NotFound:
			status = http.StatusNotFound
		case tools.Limit:
			status = http.StatusTooManyRequests
		}
		http.Error(stream.writer, "run rejected", status)
		return
	}
	if stream.messageStarted {
		_ = writeEvent(stream.writer, events.NewTextMessageEndEvent(stream.messageID))
	}
	failure := events.NewRunErrorEvent(
		"The agent could not complete this request.",
		events.WithErrorCode(string(tools.PublicCode(err))),
		events.WithRunID(stream.runID),
	)
	_ = writeEvent(stream.writer, failure)
}

func writeEvent(writer http.ResponseWriter, event events.Event) error {
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
