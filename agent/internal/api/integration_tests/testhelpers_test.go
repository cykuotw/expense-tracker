package integration_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expense-tracker/agent/internal/tools"
	"expense-tracker/agent/internal/tools/functiontool/getbalance"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"google.golang.org/adk/v2/tool"
)

const balanceRequestBody = `{"conversationId":"one","messageId":"one","text":"balance demo"}`

func decodeEvents(t *testing.T, body string) []events.Event {
	t.Helper()
	var decoded []events.Event
	for line := range strings.SplitSeq(body, "\n") {
		payload, isEvent := strings.CutPrefix(line, "data: ")
		if !isEvent {
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
	return decoded
}

func localRequest(body string) *http.Request {
	request := httptest.NewRequest(
		"POST", "http://127.0.0.1:8090/chat", strings.NewReader(body),
	)
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("Content-Type", "application/json")
	return request
}

func testCatalog(t *testing.T, client getbalance.Client) map[string]tool.Tool {
	t.Helper()
	catalog, err := tools.NewCatalog(client)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}
