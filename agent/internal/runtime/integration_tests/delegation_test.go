package integration_test

import (
	"context"
	"io/fs"
	"iter"
	"strings"
	"testing"
	"testing/fstest"
	"uuid"

	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/runtime"
	"expense-tracker/agent/internal/runtime/engine"
	"expense-tracker/agent/internal/tools"
	"expense-tracker/agent/skills"

	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
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
	app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
		Models: observation.modelFactory,
		Tools:  testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(),
		Skills:               files,
		Sessions:             store,
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(app.SkillMetadata()) != 3 {
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
	if models.PublicCode(err) != models.Limit {
		t.Fatal("duplicate accepted", err)
	}

	stored, err := store.Get(t.Context(), &session.GetRequest{
		AppName:   engine.AppName,
		UserID:    "local-fixture",
		SessionID: memorySessionID(t, store, "local-fixture", "conversation"),
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

// routingObservation records only the coordinator's declared capabilities.
type routingObservation struct {
	calls                int
	sawEchoDescription   bool
	receivedBusinessTool bool
}

func (observation *routingObservation) modelFactory(
	ctx context.Context,
	skillName string,
	modelName string,
) (model.LLM, error) {
	provider, err := fixtures.ModelFactory(ctx, skillName, modelName)
	if err != nil {
		return nil, err
	}
	return modelFunc(func(
		ctx context.Context,
		request *model.LLMRequest,
		stream bool,
	) iter.Seq2[*model.LLMResponse, error] {
		if skillName == "orchestrator" {
			observation.calls++
			observation.inspect(request)
		}
		return provider.GenerateContent(ctx, request, stream)
	}), nil
}

func (observation *routingObservation) inspect(request *model.LLMRequest) {
	for _, tool := range request.Config.Tools {
		for _, function := range tool.FunctionDeclarations {
			if function.Name == "get_balance" {
				observation.receivedBusinessTool = true
			}
			isEcho := function.Name == "skill_echo"
			if isEcho && strings.Contains(function.Description, "synthetic message") {
				observation.sawEchoDescription = true
			}
		}
	}
}

func memorySessionID(t *testing.T, store session.Service, owner, thread string) string {
	t.Helper()
	result, err := store.List(t.Context(), &session.ListRequest{AppName: engine.AppName, UserID: owner})
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Sessions) != 1 {
		t.Fatal("missing runtime thread reference")
	}
	id := result.Sessions[0].ID()
	if _, err := uuid.Parse(id); err != nil || id == thread {
		t.Fatal("session identity was not generated by the runtime")
	}
	return id
}

func bundledFiles(t *testing.T) fstest.MapFS {
	t.Helper()
	files := fstest.MapFS{}
	err := fs.WalkDir(skills.Files, ".", func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() {
			data, err := fs.ReadFile(skills.Files, path)
			if err != nil {
				return err
			}
			files[path] = &fstest.MapFile{Data: data}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}
