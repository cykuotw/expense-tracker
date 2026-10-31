package main

import (
	"encoding/json/v2"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expense-tracker/agent/config"
	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/llm"
	agentruntime "expense-tracker/agent/internal/runtime"
	"expense-tracker/agent/internal/tools"
	"expense-tracker/agent/internal/tools/functiontool/getbalance"

	"google.golang.org/adk/v2/tool"
	"google.golang.org/genai"
)

func TestFixtureStartupIsolation(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "synthetic-not-a-real-key")
	if _, err := config.Load(); err != nil {
		t.Fatal(err)
	}
	t.Setenv("AWS_LAMBDA_RUNTIME_API", "synthetic")
	if _, err := config.Load(); err == nil {
		t.Fatal("fixture accepted in Lambda")
	}
}

func TestExplicitLocalStartup(t *testing.T) {
	t.Setenv("GEMINI_API_KEY", "synthetic-not-a-real-key")
	t.Setenv("AGENT_STATE_DATABASE_URL", "synthetic-configured-dsn")
	t.Setenv("AGENT_MODE", "local")
	t.Setenv("AGENT_WORKER_URL", "http://127.0.0.1:8000")
	if _, err := config.Load(); err != nil {
		t.Fatal(err)
	}
	for _, endpoint := range []string{"", "https://remote.test", "http://0.0.0.0:8000"} {
		t.Setenv("AGENT_WORKER_URL", endpoint)
		err := run(nil)
		if err == nil || !strings.Contains(err.Error(), "AGENT_WORKER_URL") {
			t.Fatalf("expected local Worker configuration error, got %v", err)
		}
	}
	t.Setenv("AGENT_MODE", "unknown")
	if _, err := config.Load(); err == nil {
		t.Fatal("accepted unknown mode")
	}
}

func TestBuildFixtureHandler(t *testing.T) {
	// Handler construction uses injected dependencies rather than rereading env.
	t.Setenv("AGENT_MODE", "local")
	handler, err := buildHandler(t.Context(), agentruntime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
		Models: fixtures.ModelFactory, Tools: testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(),
	}, false)
	if err != nil {
		t.Fatal(err)
	}

	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8090/chat", strings.NewReader(`{"conversationId":"startup","messageId":"one","text":"hello"}`))
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "RUN_FINISHED") {
		t.Fatalf("fixture chat failed: status=%d body=%s", response.Code, response.Body.String())
	}
}

func TestRunRejectsArgumentsBeforeStartup(t *testing.T) {
	if err := run([]string{"--unexpected"}); err == nil {
		t.Fatal("accepted command-line arguments")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (send roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return send(request)
}

func TestGeminiHandlerWithOfflineProvider(t *testing.T) {
	t.Setenv("AGENT_MODE", "fixture")
	t.Setenv("GEMINI_API_KEY", "synthetic-not-a-real-key")
	t.Setenv("AGENT_DEFAULT_MODEL", "operator-default")
	t.Setenv("AGENT_ROUTER_MODEL", "operator-router")
	t.Setenv("AGENT_ROUTER_MAX_OUTPUT_TOKENS", "29")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		if !strings.Contains(request.URL.Path, "streamGenerateContent") {
			t.Fatalf("unexpected Gemini operation: %s", request.URL.Path)
		}
		if !strings.Contains(request.URL.Path, "operator-router") {
			t.Fatal("configured router model did not reach the provider")
		}
		payload, err := io.ReadAll(request.Body)
		if err != nil {
			t.Fatal(err)
		}
		var body struct {
			GenerationConfig struct {
				MaxOutputTokens int32 `json:"maxOutputTokens"`
			} `json:"generationConfig"`
		}
		if err := json.Unmarshal(payload, &body); err != nil {
			t.Fatal(err)
		}
		if body.GenerationConfig.MaxOutputTokens != 29 {
			t.Fatal("configured token ceiling did not reach the provider")
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": {"text/event-stream"}},
			Body:       io.NopCloser(strings.NewReader("data: {\"candidates\":[{\"content\":{\"role\":\"model\",\"parts\":[{\"text\":\"Hello from Gemini.\"}]},\"finishReason\":\"STOP\"}]}\n\n")),
		}, nil
	})}
	handler, err := buildHandler(t.Context(), agentruntime.Dependencies{ModelPolicy: cfg.ModelPolicy, Limits: cfg.Limits,
		Models: llm.NewGeminiFactory(genai.ClientConfig{
			APIKey: "synthetic-not-a-real-key", Backend: genai.BackendGeminiAPI, HTTPClient: client,
		}),
		Tools: testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
		RouterTools: tools.RouterTools(), InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks: tools.AfterModelCallbacks(),
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if calls != 0 {
		t.Fatal("handler construction performed model inference")
	}
	request := httptest.NewRequest(http.MethodPost, "http://127.0.0.1:8090/chat", strings.NewReader(`{"conversationId":"gemini","messageId":"one","text":"hello"}`))
	request.RemoteAddr = "127.0.0.1:12345"
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if calls != 1 || response.Code != http.StatusOK || !strings.Contains(response.Body.String(), "Hello from Gemini.") || !strings.Contains(response.Body.String(), "RUN_FINISHED") {
		t.Fatalf("Gemini chat failed: calls=%d status=%d body=%s", calls, response.Code, response.Body.String())
	}
}

func testCatalog(t *testing.T, client getbalance.Client) map[string]tool.Tool {
	t.Helper()
	catalog, err := tools.NewCatalog(client)
	if err != nil {
		t.Fatal(err)
	}
	return catalog
}
