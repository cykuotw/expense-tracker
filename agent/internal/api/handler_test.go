package api

import (
	"bufio"
	"context"
	"errors"
	"io"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expense-tracker/agent/internal/models"
	agentruntime "expense-tracker/agent/internal/runtime"
	"expense-tracker/agent/internal/tools"
	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"google.golang.org/adk/v2/model"
)

const balanceRequestBody = `{"conversationId":"one","messageId":"one","text":"balance demo"}`

type gatedModel struct {
	model.LLM
	gate <-chan struct{}
}

func (provider gatedModel) GenerateContent(
	ctx context.Context,
	request *model.LLMRequest,
	stream bool,
) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		paused := false
		for response, err := range provider.LLM.GenerateContent(ctx, request, stream) {
			if !yield(response, err) {
				return
			}
			if response != nil && response.Partial && !paused {
				paused = true
				select {
				case <-provider.gate:
				case <-ctx.Done():
					return
				}
			}
		}
	}
}

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

// A gated model cannot produce its second delta until the HTTP client has read
// the first. This verifies socket delivery, rather than recorder Flush flags.
func TestIncrementalFixtureSSE(t *testing.T) {
	gate := make(chan struct{})
	factory := func(ctx context.Context, skillName, modelName string) (model.LLM, error) {
		provider, err := models.FixtureFactory(ctx, skillName, modelName)
		if skillName == "orchestrator" {
			provider = gatedModel{LLM: provider, gate: gate}
		}
		return provider, err
	}
	application, err := agentruntime.New(t.Context(), agentruntime.Config{
		Models:  factory,
		Balance: tools.FixtureBalanceClient{},
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(NewLocalFixture(application))
	defer server.Close()

	ctx, cancel := context.WithTimeout(t.Context(), 5*time.Second)
	defer cancel()

	const payload = `{"conversationId":"conversation","messageId":"one","text":"balance demo"}`
	request, err := http.NewRequestWithContext(
		ctx, "POST", server.URL+"/chat", strings.NewReader(payload),
	)
	if err != nil {
		t.Fatal(err)
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := server.Client().Do(request)
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()

	validHeaders := response.StatusCode == http.StatusOK &&
		response.Header.Get("Content-Type") == "text/event-stream" &&
		response.Header.Get("Cache-Control") == "no-store"
	if !validHeaders {
		t.Fatal(response.Status, response.Header)
	}

	scanner := bufio.NewScanner(response.Body)
	var body strings.Builder
	firstDelta := false
	for scanner.Scan() {
		line := scanner.Text()
		body.WriteString(line)
		body.WriteByte('\n')
		if strings.Contains(line, `"TEXT_MESSAGE_CONTENT"`) {
			firstDelta = true
			break
		}
	}
	close(gate)
	if !firstDelta {
		t.Fatal("first delta not delivered while model was blocked", scanner.Err())
	}
	for scanner.Scan() {
		body.WriteString(scanner.Text())
		body.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		t.Fatal(err)
	}

	decoded := decodeEvents(t, body.String())
	expected := []events.EventType{
		events.EventTypeRunStarted,
		events.EventTypeTextMessageStart,
		events.EventTypeTextMessageContent,
		events.EventTypeTextMessageContent,
		events.EventTypeTextMessageEnd,
		events.EventTypeRunFinished,
	}
	if len(decoded) != len(expected) {
		t.Fatalf("unexpected SSE sequence: %s", body.String())
	}
	for index, event := range decoded {
		if event.Type() != expected[index] {
			t.Fatalf("event %d: %s", index, event.Type())
		}
	}

	hasBalance := strings.Contains(body.String(), "12.34")
	hasSingleAnswer := strings.Count(body.String(), "Synthetic fixture balance:") == 1
	if !hasBalance || !hasSingleAnswer {
		t.Fatal(body.String())
	}

	start := decoded[0].(*events.RunStartedEvent)
	startPayload, err := decoded[1].ToJSON()
	if err != nil || !strings.Contains(string(startPayload), `"role":"assistant"`) {
		t.Fatal("assistant role missing", string(startPayload), err)
	}
	finish := decoded[len(decoded)-1].(*events.RunFinishedEvent)
	sameRun := start.RunID() != "" && start.RunID() == finish.RunID()
	if !sameRun || start.ThreadID() != "conversation" {
		t.Fatal("invalid run correlation")
	}
	for _, privateField := range []string{
		"LLMResponse", "skillVersion", "FunctionResponse", "thoughtSignature",
	} {
		if strings.Contains(body.String(), privateField) {
			t.Fatal("private data escaped", privateField)
		}
	}
}

type failingApplication struct{}

func (failingApplication) Run(
	_ context.Context,
	request agentruntime.Request,
	emit func(agentruntime.Output) error,
) error {
	start := agentruntime.Output{
		Kind:           "started",
		RunID:          "server-run",
		ConversationID: request.ConversationID,
	}
	if err := emit(start); err != nil {
		return err
	}
	if err := emit(agentruntime.Output{Kind: "text", Delta: "Partial answer"}); err != nil {
		return err
	}
	return errors.New("private provider credential and original reasoning")
}

func localRequest(body string) *http.Request {
	request := httptest.NewRequest(
		"POST", "http://127.0.0.1:8090/chat", strings.NewReader(body),
	)
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("Content-Type", "application/json")
	return request
}

func TestTerminalErrorAfterStreaming(t *testing.T) {
	writer := httptest.NewRecorder()
	NewLocalFixture(failingApplication{}).ServeHTTP(writer, localRequest(balanceRequestBody))
	if writer.Code != http.StatusOK {
		t.Fatal(writer.Code)
	}

	decoded := decodeEvents(t, writer.Body.String())
	if decoded[len(decoded)-1].Type() != events.EventTypeRunError {
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

func TestValidationAndFixtureIsolation(t *testing.T) {
	testCases := []struct {
		name   string
		body   string
		mutate func(*http.Request)
		status int
	}{
		{
			name:   "invalid json",
			body:   "{}",
			status: http.StatusBadRequest,
		},
		{
			name:   "trailing json",
			body:   balanceRequestBody + "{}",
			status: http.StatusBadRequest,
		},
		{
			name:   "unknown field",
			body:   `{"conversationId":"one","messageId":"one","text":"x","actor":"forged"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "duplicate key",
			body:   `{"conversationId":"one","conversationId":"two","messageId":"one","text":"x"}`,
			status: http.StatusBadRequest,
		},
		{
			name:   "oversized",
			body:   strings.Repeat("x", 17<<10),
			status: http.StatusRequestEntityTooLarge,
		},
		{
			name: "remote peer",
			body: balanceRequestBody,
			mutate: func(request *http.Request) {
				request.RemoteAddr = "192.0.2.1:1234"
			},
			status: http.StatusForbidden,
		},
		{
			name: "foreign host",
			body: balanceRequestBody,
			mutate: func(request *http.Request) {
				request.Host = "foreign.example"
				request.URL.Host = "foreign.example"
			},
			status: http.StatusForbidden,
		},
		{
			name: "origin",
			body: balanceRequestBody,
			mutate: func(request *http.Request) {
				request.Header.Set("Origin", "https://foreign.example")
			},
			status: http.StatusForbidden,
		},
		{
			name: "forged identity",
			body: balanceRequestBody,
			mutate: func(request *http.Request) {
				request.Header.Set("X-Actor-ID", "admin")
			},
			status: http.StatusForbidden,
		},
		{
			name: "bearer",
			body: balanceRequestBody,
			mutate: func(request *http.Request) {
				request.Header.Set("Authorization", "Bearer forged")
			},
			status: http.StatusForbidden,
		},
		{
			name: "cookie",
			body: balanceRequestBody,
			mutate: func(request *http.Request) {
				request.Header.Set("Cookie", "session=forged")
			},
			status: http.StatusForbidden,
		},
		{
			name: "content type",
			body: balanceRequestBody,
			mutate: func(request *http.Request) {
				request.Header.Set("Content-Type", "text/plain")
			},
			status: http.StatusUnsupportedMediaType,
		},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			request := localRequest(testCase.body)
			if testCase.mutate != nil {
				testCase.mutate(request)
			}
			writer := httptest.NewRecorder()
			NewLocalFixture(failingApplication{}).ServeHTTP(writer, request)
			if writer.Code != testCase.status {
				t.Fatalf("status %d want %d: %s", writer.Code, testCase.status, writer.Body.String())
			}
			if strings.Contains(writer.Body.String(), "RUN_STARTED") {
				t.Fatal("invalid request started model work")
			}
		})
	}
}

func TestClientDisconnectStopsRun(t *testing.T) {
	ctx, cancel := context.WithCancel(t.Context())
	cancel()

	request := localRequest(balanceRequestBody).WithContext(ctx)
	writer := httptest.NewRecorder()
	application, err := agentruntime.New(t.Context(), agentruntime.Config{
		Models:  models.FixtureFactory,
		Balance: tools.FixtureBalanceClient{},
	})
	if err != nil {
		t.Fatal(err)
	}
	NewLocalFixture(application).ServeHTTP(writer, request)
	if strings.Contains(writer.Body.String(), "RUN_FINISHED") {
		t.Fatal("cancelled run reported success")
	}
	_, _ = io.Copy(io.Discard, writer.Result().Body)
}
