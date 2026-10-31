package integration_test

import (
	"bufio"
	"context"
	"iter"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expense-tracker/agent/internal/api"
	"expense-tracker/agent/internal/fixtures"
	agentruntime "expense-tracker/agent/internal/runtime"
	"expense-tracker/agent/internal/tools"

	"github.com/ag-ui-protocol/ag-ui/sdks/community/go/pkg/core/events"
	"google.golang.org/adk/v2/model"
)

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

// A gated model cannot produce the second delta until the client reads the first.
func TestIncrementalFixtureSSE(t *testing.T) {
	gate := make(chan struct{})
	factory := func(ctx context.Context, skillName, modelName string) (model.LLM, error) {
		provider, err := fixtures.ModelFactory(ctx, skillName, modelName)
		if skillName == "orchestrator" {
			provider = gatedModel{LLM: provider, gate: gate}
		}
		return provider, err
	}
	application, err := agentruntime.New(t.Context(), agentruntime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
		Models: factory,
		Tools:  testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(),
	})
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewServer(api.NewLocalFixtureHandler(application))
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
