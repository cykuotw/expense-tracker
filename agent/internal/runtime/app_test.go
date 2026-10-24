package runtime

import (
	"context"
	"iter"
	"strings"
	"sync"
	"testing"
	"testing/fstest"

	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/tools"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

func TestDiscoveryClarificationHandbackAndTaskSwitching(t *testing.T) {
	files := bundledFiles(t)
	files["echo/SKILL.md"] = &fstest.MapFile{
		Data: []byte(`---
name: echo
description: Echo a synthetic message.
---
Return a synthetic echo using finish_task.
`),
	}
	store := session.InMemoryService()
	observation := new(routingObservation)
	app, err := New(t.Context(), Config{
		Models:   observation.modelFactory,
		Balance:  tools.FixtureBalanceClient{},
		Skills:   files,
		Sessions: store,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(app.Diagnostics()) != 3 {
		t.Fatal("leaf not discovered")
	}
	runIDs := map[string]bool{}
	for _, testCase := range []struct {
		messageID string
		text      string
		want      string
	}{
		{"one", "balance", "Which group"},
		{"two", "demo", "12.34"},
		{"three", "echo hello", "echo hello"},
		{"four", "balance demo", "12.34"},
		{"five", "unsupported request", "I can check"},
	} {
		outputs, text, err := runMessage(t, app, "conversation", testCase.messageID, testCase.text)
		if err != nil {
			t.Fatalf("%s: %v", testCase.messageID, err)
		}
		if !strings.Contains(text, testCase.want) {
			t.Fatalf("%s: got %q want %q", testCase.messageID, text, testCase.want)
		}
		if strings.Count(text, "Synthetic fixture balance:") > 1 {
			t.Fatal("duplicate final output", text)
		}
		if outputs[0].Kind != "started" || outputs[len(outputs)-1].Kind != "finished" {
			t.Fatal("invalid lifecycle")
		}
		runID := outputs[0].RunID
		if runID == "" || runIDs[runID] {
			t.Fatal("run identity reused")
		}
		runIDs[runID] = true
		for _, output := range outputs {
			if output.RunID != runID || output.ClientMessageID != testCase.messageID {
				t.Fatal("lost correlation")
			}
			missingMetadata := output.InvocationID == "" ||
				output.SkillVersion == "" ||
				output.Model == ""
			if output.Kind == "text" && missingMetadata {
				t.Fatal("missing execution metadata")
			}
		}
	}
	if observation.calls < 6 || !observation.sawEchoDescription {
		t.Fatal("missing coordinator handback or discovered description", observation.calls)
	}
	if observation.receivedBusinessTool {
		t.Error("root received business tool")
	}
	_, _, err = runMessage(t, app, "conversation", "one", "balance demo")
	if tools.PublicCode(err) != tools.Limit {
		t.Fatal("duplicate accepted", err)
	}

	stored, err := store.Get(t.Context(), &session.GetRequest{
		AppName:   sessionAppName,
		UserID:    "local-fixture",
		SessionID: "conversation",
	})
	if err != nil {
		t.Fatal(err)
	}
	finishes := 0
	balanceReads := 0
	for event := range stored.Session.Events().All() {
		if event.Content == nil {
			continue
		}
		for _, part := range event.Content.Parts {
			if part.FunctionCall != nil && part.FunctionCall.Name == "get_balance" {
				balanceReads++
			}
			if part.FunctionCall != nil && part.FunctionCall.Name == "finish_task" {
				finishes++
			}
		}
	}
	if finishes != 3 {
		t.Fatalf("expected three completed tasks, got %d", finishes)
	}
	if balanceReads != 2 {
		t.Fatalf("balance tasks reused an old result: %d reads", balanceReads)
	}

	// Another conversation must not continue this conversation's balance task.
	_, text, err := runMessage(t, app, "other-conversation", "one", "demo")
	if err != nil || strings.Contains(text, "12.34") {
		t.Fatal("session leaked", text, err)
	}
}

func TestInvalidSkillsAndGrantsFailConstruction(t *testing.T) {
	for _, metadata := range []string{
		"type: sequential", "type: loop", "type: unknown",
		"mode: task", "allowed-tools: delete_expense", "allowed-tools: get_balance get_balance",
	} {
		t.Run(metadata, func(t *testing.T) {
			files := bundledFiles(t)
			files["extra/SKILL.md"] = &fstest.MapFile{Data: []byte(
				"---\nname: extra\ndescription: Extra capability.\nmetadata:\n  " + metadata + "\n---\nInstructions.\n",
			)}
			_, err := New(t.Context(), Config{
				Models:  models.FixtureFactory,
				Balance: tools.FixtureBalanceClient{},
				Skills:  files,
			})
			if err == nil {
				t.Fatal("invalid skill accepted")
			}
		})
	}
	for _, remove := range []string{"orchestrator/SKILL.md", "get-balance/SKILL.md"} {
		files := bundledFiles(t)
		delete(files, remove)
		if _, err := New(t.Context(), Config{
			Models:  models.FixtureFactory,
			Balance: tools.FixtureBalanceClient{},
			Skills:  files,
		}); err == nil {
			t.Fatal("incomplete registry accepted")
		}
	}
	files := bundledFiles(t)
	content := string(files["orchestrator/SKILL.md"].Data)
	modified := strings.Replace(content, "metadata:", "metadata:\n  allowed-tools: get_balance", 1)
	files["orchestrator/SKILL.md"] = &fstest.MapFile{Data: []byte(modified)}
	if _, err := New(t.Context(), Config{
		Models:  models.FixtureFactory,
		Balance: tools.FixtureBalanceClient{},
		Skills:  files,
	}); err == nil {
		t.Fatal("root business grant accepted")
	}

	for _, request := range []Request{
		{ConversationID: "../x", MessageID: "one", Text: "text"},
		{ConversationID: "a", MessageID: "one", Text: strings.Repeat("x", MaxTextBytes+1)},
		{ConversationID: "a", MessageID: "one", Text: " "},
	} {
		if request.Validate() == nil {
			t.Fatal("invalid request accepted")
		}
	}
}

func TestFunctionToolFailureDoesNotProduceBalance(t *testing.T) {
	app, err := New(t.Context(), Config{
		Models:  models.FixtureFactory,
		Balance: unavailableBalance{},
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
			app, err := New(t.Context(), Config{
				Models:  failingModelFactory(scenario),
				Balance: tools.FixtureBalanceClient{},
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

func TestConcurrentRunLimitsAndRelease(t *testing.T) {
	entered := make(chan struct{}, 4)
	release := make(chan struct{})
	factory := func(context.Context, string, string) (model.LLM, error) {
		return modelFunc(func(
			ctx context.Context,
			_ *model.LLMRequest,
			_ bool,
		) iter.Seq2[*model.LLMResponse, error] {
			return func(yield func(*model.LLMResponse, error) bool) {
				entered <- struct{}{}
				select {
				case <-release:
				case <-ctx.Done():
					yield(nil, ctx.Err())
					return
				}
				yield(&model.LLMResponse{
					Content:      genai.NewContentFromText(`{"kind":"unsupported"}`, "model"),
					TurnComplete: true,
				}, nil)
			}
		}), nil
	}
	app, err := New(t.Context(), Config{
		Models:  factory,
		Balance: tools.FixtureBalanceClient{},
	})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 4)
	for _, id := range []string{"one", "two", "three", "four"} {
		wg.Go(func() {
			results <- app.Run(t.Context(), Request{
				ConversationID: id,
				MessageID:      "one",
				Text:           "unsupported",
			}, func(Output) error { return nil })
		})
	}
	for range 4 {
		<-entered
	}
	for _, id := range []string{"one", "five"} {
		err := app.Run(t.Context(), Request{
			ConversationID: id,
			MessageID:      "two",
			Text:           "unsupported",
		}, func(Output) error { return nil })
		if tools.PublicCode(err) != tools.Limit {
			t.Error("concurrent run was accepted", err)
		}
	}
	close(release)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err = app.Run(t.Context(), Request{
		ConversationID: "one",
		MessageID:      "two",
		Text:           "unsupported",
	}, func(Output) error { return nil }); err != nil {
		t.Fatal("run slot not released", err)
	}
}
