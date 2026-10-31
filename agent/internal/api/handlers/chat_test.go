package handlers

import (
	"context"
	"errors"
	"expense-tracker/agent/internal/models"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

type failingApplication struct {
	Application
}

func (failingApplication) Run(_ context.Context, request models.ChatRequest, emit func(models.Output) error) error {
	for _, output := range []models.Output{
		{Kind: "started", RunID: "server-run", ConversationID: request.ConversationID},
		{Kind: "text", Delta: "Partial answer"},
	} {
		if err := emit(output); err != nil {
			return err
		}
	}
	return errors.New("private provider credential and original reasoning")
}

func TestTerminalErrorAfterStreaming(t *testing.T) {
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8090/chat", strings.NewReader(`{"conversationId":"one","messageId":"one","text":"balance demo"}`))
	request.Header.Set("Content-Type", "application/json")
	writer := httptest.NewRecorder()
	New(failingApplication{}).HandleChat(writer, request)
	if writer.Code != http.StatusOK {
		t.Fatal(writer.Code)
	}
	var decoded []events.Event
	for line := range strings.SplitSeq(writer.Body.String(), "\n") {
		payload, ok := strings.CutPrefix(line, "data: ")
		if !ok {
			continue
		}
		event, err := events.EventFromJSON([]byte(payload))
		if err != nil {
			t.Fatal(err)
		}
		if err := event.Validate(); err != nil {
			t.Fatal(err)
		}
		decoded = append(decoded, event)
	}
	if err := events.ValidateSequence(decoded); err != nil {
		t.Fatal(err)
	}
	if len(decoded) == 0 || decoded[len(decoded)-1].Type() != events.EventTypeRunError {
		t.Fatal(writer.Body.String())
	}
	if strings.Contains(writer.Body.String(), "private") {
		t.Fatal(writer.Body.String())
	}
	for _, event := range decoded {
		if event.Type() == events.EventTypeRunFinished {
			t.Fatal("failed run marked successful")
		}
	}
}
