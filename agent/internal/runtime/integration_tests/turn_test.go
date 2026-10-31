package integration_test

import (
	"bytes"
	"context"
	"encoding/json/v2"
	"errors"
	"iter"
	"log/slog"
	"strings"
	"sync/atomic"
	"testing"

	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/llm"
	"expense-tracker/agent/internal/runtime"
	"expense-tracker/agent/internal/tools"
	"expense-tracker/agent/internal/tools/functiontool/getbalance"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

func TestFunctionToolFailureDoesNotProduceBalance(t *testing.T) {
	app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
		Models: fixtures.ModelFactory,
		Tools:  testCatalog(t, unavailableBalance{}), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, text, err := runMessage(t, app, "one", "one", "balance demo")
	if err == nil || strings.Contains(err.Error(), "private") || text != "" {
		t.Fatal("tool failure escaped", text, err)
	}
}

func TestInvalidDelegationProviderFailureAndPrematureFinish(t *testing.T) {
	for _, scenario := range []string{"provider", "unknown-delegation", "premature-finish"} {
		t.Run(scenario, func(t *testing.T) {
			app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
				Models: failingModelFactory(scenario),
				Tools:  testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
				RouterTools:          tools.RouterTools(),
				InstructionProviders: tools.InstructionProviders(),
				AfterModelCallbacks:  tools.AfterModelCallbacks(),
			})
			if err != nil {
				t.Fatal(err)
			}
			_, text, err := runMessage(t, app, "one", "one", "balance demo")
			if err == nil || text != "" || strings.Contains(err.Error(), "private") {
				t.Fatal("controlled failure missing", text, err)
			}
		})
	}
}

func TestBalanceCompletionCannotReuseAnEarlierInvocation(t *testing.T) {
	var premature atomic.Bool
	factory := func(ctx context.Context, skill, name string) (model.LLM, error) {
		base, err := fixtures.ModelFactory(ctx, skill, name)
		require.NoError(t, err)
		invalid, err := failingModelFactory("premature-finish")(ctx, skill, name)
		require.NoError(t, err)
		return modelFunc(func(ctx context.Context, request *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
			if skill == "get-balance" && premature.Load() {
				return invalid.GenerateContent(ctx, request, stream)
			}
			return base.GenerateContent(ctx, request, stream)
		}), nil
	}
	app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
		Models: factory, Tools: testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(),
	})
	require.NoError(t, err)
	_, answer, err := runMessage(t, app, "thread", "one", "balance demo")
	require.NoError(t, err)
	require.Contains(t, answer, "12.34")
	premature.Store(true)
	_, answer, err = runMessage(t, app, "thread", "two", "balance demo")
	require.Error(t, err)
	require.Empty(t, answer, "previous reads must not authorize a fabricated answer in a new invocation")
}

type delegationLog struct {
	Message        string `json:"msg"`
	ConversationID string `json:"conversationId"`
	MessageID      string `json:"messageId"`
	RunID          string `json:"runId"`
	InvocationID   string `json:"invocationId"`
	CallID         string `json:"callId"`
	From           string `json:"from"`
	To             string `json:"to"`
}

func readDelegationLogs(t *testing.T, buffer *bytes.Buffer) []delegationLog {
	t.Helper()
	var entries []delegationLog
	for line := range strings.SplitSeq(strings.TrimSpace(buffer.String()), "\n") {
		if line == "" {
			continue
		}
		var entry delegationLog
		if err := json.Unmarshal([]byte(line), &entry); err != nil {
			t.Fatal(err)
		}
		entries = append(entries, entry)
	}
	return entries
}

func TestDelegationLogsAcrossClarificationAndCompletion(t *testing.T) {
	var buffer bytes.Buffer
	app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
		Models: fixtures.ModelFactory,
		Tools:  testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(),
		Logger:               slog.New(slog.NewJSONHandler(&buffer, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}

	first, _, err := runMessage(t, app, "conversation", "one", "balance PRIVATE_PROMPT_SENTINEL")
	if err != nil {
		t.Fatal(err)
	}
	entries := readDelegationLogs(t, &buffer)
	if len(entries) != 1 || entries[0].Message != "agent handoff" {
		t.Fatalf("clarification must log a handoff without handback: %+v", entries)
	}
	handoff := entries[0]
	if handoff.From != "orchestrator" || handoff.To != "get-balance" {
		t.Fatal("incorrect handoff target", handoff)
	}
	if handoff.RunID != first[0].RunID || handoff.MessageID != "one" {
		t.Fatal("lost originating run correlation", handoff)
	}

	second, _, err := runMessage(t, app, "conversation", "two", "demo")
	if err != nil {
		t.Fatal(err)
	}
	entries = readDelegationLogs(t, &buffer)
	if len(entries) != 2 || entries[1].Message != "agent handback" {
		t.Fatalf("resumed task must log handback without another handoff: %+v", entries)
	}
	handback := entries[1]
	if handback.From != "get-balance" || handback.To != "orchestrator" {
		t.Fatal("incorrect handback target", handback)
	}
	if handback.RunID != second[0].RunID || handback.MessageID != "two" {
		t.Fatal("lost completion run correlation", handback)
	}
	if handoff.CallID == "" || handoff.CallID != handback.CallID {
		t.Fatal("cross-run task correlation lost", entries)
	}
	for _, entry := range entries {
		if entry.ConversationID != "conversation" || entry.InvocationID == "" {
			t.Fatal("missing execution correlation", entry)
		}
	}

	_, _, err = runMessage(t, app, "conversation", "three", "balance demo")
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = runMessage(t, app, "conversation", "four", "unsupported")
	if err != nil {
		t.Fatal(err)
	}
	entries = readDelegationLogs(t, &buffer)
	if len(entries) != 4 || entries[2].Message != "agent handoff" || entries[3].Message != "agent handback" {
		t.Fatal("expected one log pair for the fresh task and none for unsupported intent", entries)
	}
	for _, excluded := range []string{
		"PRIVATE_PROMPT_SENTINEL", "12.34", "Which group", "groupId", "get_balance",
	} {
		if strings.Contains(buffer.String(), excluded) {
			t.Fatal("interaction content escaped into logs", excluded)
		}
	}
}

func TestFailedTaskDoesNotLogSuccessfulHandback(t *testing.T) {
	var buffer bytes.Buffer
	app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
		Models: fixtures.ModelFactory,
		Tools:  testCatalog(t, unavailableBalance{}), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(),
		Logger:               slog.New(slog.NewJSONHandler(&buffer, nil)),
	})
	if err != nil {
		t.Fatal(err)
	}
	_, _, err = runMessage(t, app, "conversation", "one", "balance demo")
	if err == nil {
		t.Fatal("fixture should fail")
	}
	entries := readDelegationLogs(t, &buffer)
	if len(entries) != 1 || entries[0].Message != "agent handoff" {
		t.Fatal("failed task logged successful handback", entries)
	}
	if strings.Contains(buffer.String(), "private backend error") {
		t.Fatal("raw error logged")
	}
}

type unavailableBalance struct{}

func (unavailableBalance) GetBalance(context.Context, string) (getbalance.Result, error) {
	return getbalance.Result{}, errors.New("private backend error")
}

func failingModelFactory(scenario string) llm.Factory {
	return func(ctx context.Context, skillName, modelName string) (model.LLM, error) {
		if scenario == "premature-finish" && skillName == "orchestrator" {
			return fixtures.ModelFactory(ctx, skillName, modelName)
		}
		return modelFunc(func(
			context.Context,
			*model.LLMRequest,
			bool,
		) iter.Seq2[*model.LLMResponse, error] {
			return func(yield func(*model.LLMResponse, error) bool) {
				if scenario == "provider" {
					yield(nil, errors.New("private provider failure"))
					return
				}

				functionName := "unknown_agent"
				arguments := map[string]any{"request": "balance demo"}
				if scenario == "premature-finish" {
					functionName = "finish_task"
					arguments = map[string]any{"result": "fabricated balance"}
				}
				yield(&model.LLMResponse{
					Content: &genai.Content{
						Role: "model",
						Parts: []*genai.Part{
							{
								FunctionCall: &genai.FunctionCall{
									Name: functionName,
									Args: arguments,
								},
							},
						},
					},
				}, nil)
			}
		}), nil
	}
}
