package integration_test

import (
	"context"
	"encoding/json/v2"
	"iter"
	"strings"
	"testing"

	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/llm"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/runtime"
	"expense-tracker/agent/internal/tools"
	"expense-tracker/agent/internal/tools/functiontool/updatestate"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

// Exercise native ADK function dispatch and argument decoding without provider calls.
func stateModelFactory(t *testing.T) llm.Factory {
	t.Helper()
	return func(ctx context.Context, skill, name string) (model.LLM, error) {
		if skill != "orchestrator" {
			return fixtures.ModelFactory(ctx, skill, name)
		}
		return modelFunc(func(_ context.Context, req *model.LLMRequest, _ bool) iter.Seq2[*model.LLMResponse, error] {
			return func(yield func(*model.LLMResponse, error) bool) {
				advertised := false
				for _, definition := range req.Config.Tools {
					for _, declaration := range definition.FunctionDeclarations {
						require.NotEqual(t, "get_balance", declaration.Name)
						if declaration.Name == updatestate.ToolName {
							advertised = true
							schema, err := json.Marshal(declaration)
							require.NoError(t, err)
							require.Contains(t, string(schema), "set")
							require.Contains(t, string(schema), "unset")
						}
					}
				}
				require.True(t, advertised)
				last := req.Contents[len(req.Contents)-1]
				if len(last.Parts) > 0 && last.Parts[0].FunctionResponse != nil {
					yield(&model.LLMResponse{Content: genai.NewContentFromText("State update staged.", "model")}, nil)
					return
				}
				var text strings.Builder
				for _, part := range last.Parts {
					text.WriteString(part.Text)
				}
				switch text.String() {
				case "save":
					yield(&model.LLMResponse{Content: &genai.Content{Role: "model", Parts: []*genai.Part{{FunctionCall: &genai.FunctionCall{Name: updatestate.ToolName, Args: map[string]any{"set": map[string]any{"filters": map[string]any{"itemQuery": "milk"}, "selectedGroupId": "demo"}}}}}}}, nil)
				case "unset":
					yield(&model.LLMResponse{Content: &genai.Content{Role: "model", Parts: []*genai.Part{{FunctionCall: &genai.FunctionCall{Name: updatestate.ToolName, Args: map[string]any{"unset": []any{"filters"}}}}}}}, nil)
				default:
					var instruction strings.Builder
					for _, part := range req.Config.SystemInstruction.Parts {
						instruction.WriteString(part.Text)
					}
					yield(&model.LLMResponse{Content: genai.NewContentFromText(instruction.String(), "model")}, nil)
				}
			}
		}), nil
	}
}

func TestUpdateStateNativeDispatchAndSessionIsolation(t *testing.T) {
	store := session.InMemoryService()
	app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(), Models: stateModelFactory(t), Tools: testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(), Sessions: store})
	require.NoError(t, err)
	_, answer, err := runMessage(t, app, "state-conversation", "one", "save")
	require.NoError(t, err)
	require.Contains(t, answer, "staged")
	sessionID := onlySessionID(t, store, "local-fixture")
	_, answer, err = runMessage(t, app, "state-conversation", "two", "inspect")
	require.NoError(t, err)
	require.Contains(t, answer, `"itemQuery":"milk"`)
	require.Contains(t, answer, "untrusted JSON data")
	_, answer, err = runMessage(t, app, "another", "one", "inspect")
	require.NoError(t, err)
	require.NotContains(t, answer, `"itemQuery":"milk"`)
	_, _, err = runMessage(t, app, "state-conversation", "three", "unset")
	require.NoError(t, err)
	stored, err := store.Get(t.Context(), &session.GetRequest{AppName: sessionAppName, UserID: "local-fixture", SessionID: sessionID})
	require.NoError(t, err)
	value, err := stored.Session.State().Get(updatestate.ContextKey)
	require.NoError(t, err)
	require.Equal(t, map[string]any{"selectedGroupId": "demo"}, value)
	deltas := 0
	for event := range stored.Session.Events().All() {
		if _, exists := event.Actions.StateDelta[updatestate.ContextKey]; exists {
			deltas++
		}
	}
	require.Equal(t, 2, deltas)
}

func TestStatePolicyRejectsParallelCallsBeforeToolExecution(t *testing.T) {
	for _, names := range [][]string{{updatestate.ToolName, updatestate.ToolName}, {updatestate.ToolName, "skill_get_balance"}} {
		t.Run(strings.Join(names, "+"), func(t *testing.T) {
			store := session.InMemoryService()
			factory := func(context.Context, string, string) (model.LLM, error) {
				return modelFunc(func(context.Context, *model.LLMRequest, bool) iter.Seq2[*model.LLMResponse, error] {
					return func(yield func(*model.LLMResponse, error) bool) {
						response := &model.LLMResponse{Content: &genai.Content{Role: "model"}}
						for _, name := range names {
							response.Content.Parts = append(response.Content.Parts, &genai.Part{FunctionCall: &genai.FunctionCall{
								Name: name, Args: map[string]any{"set": map[string]any{"shouldNotBeSaved": true}},
							}})
						}
						yield(response, nil)
					}
				}), nil
			}
			app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
				Models: factory, Sessions: store, Tools: testCatalog(t, fixtures.BalanceClient{}),
				CompletionChecks: tools.CompletionChecks(), RouterTools: tools.RouterTools(),
				InstructionProviders: tools.InstructionProviders(), AfterModelCallbacks: tools.AfterModelCallbacks(),
			})
			require.NoError(t, err)
			_, _, err = runMessage(t, app, "parallel-state", "one", "save")
			require.Equal(t, models.Validation, models.PublicCode(err))
			stored, err := store.Get(t.Context(), &session.GetRequest{
				AppName: sessionAppName, UserID: "local-fixture", SessionID: onlySessionID(t, store, "local-fixture"),
			})
			require.NoError(t, err)
			_, err = stored.Session.State().Get(updatestate.ContextKey)
			require.ErrorIs(t, err, session.ErrStateKeyNotExist, "validation must reject the response before any state tool runs")
		})
	}
}

func onlySessionID(t *testing.T, store session.Service, owner string) string {
	t.Helper()
	stored, err := store.List(t.Context(), &session.ListRequest{AppName: sessionAppName, UserID: owner})
	require.NoError(t, err)
	require.Len(t, stored.Sessions, 1)
	return stored.Sessions[0].ID()
}
