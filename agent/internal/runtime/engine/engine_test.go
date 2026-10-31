package engine

import (
	"strings"
	"testing"
	"testing/fstest"

	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/tools"
)

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
			_, err := New(t.Context(), Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
				Models: fixtures.ModelFactory,
				Tools:  testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
				RouterTools:          tools.RouterTools(),
				InstructionProviders: tools.InstructionProviders(),
				AfterModelCallbacks:  tools.AfterModelCallbacks(),
				Skills:               files,
			})
			if err == nil {
				t.Fatal("invalid skill accepted")
			}
		})
	}
	for _, remove := range []string{"orchestrator/SKILL.md", "get-balance/SKILL.md"} {
		files := bundledFiles(t)
		delete(files, remove)
		if _, err := New(t.Context(), Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
			Models: fixtures.ModelFactory,
			Tools:  testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
			RouterTools:          tools.RouterTools(),
			InstructionProviders: tools.InstructionProviders(),
			AfterModelCallbacks:  tools.AfterModelCallbacks(),
			Skills:               files,
		}); err == nil {
			t.Fatal("incomplete registry accepted")
		}
	}
	files := bundledFiles(t)
	content := string(files["orchestrator/SKILL.md"].Data)
	modified := strings.Replace(content, "allowed-tools: update_state", "allowed-tools: get_balance", 1)
	files["orchestrator/SKILL.md"] = &fstest.MapFile{Data: []byte(modified)}
	if _, err := New(t.Context(), Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
		Models: fixtures.ModelFactory,
		Tools:  testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(),
		Skills:               files,
	}); err == nil {
		t.Fatal("root business grant accepted")
	}

	for _, request := range []models.ChatRequest{
		{ConversationID: "../x", MessageID: "one", Text: "text"},
		{ConversationID: "a", MessageID: "one", Text: strings.Repeat("x", models.MaxTextBytes+1)},
		{ConversationID: "a", MessageID: "one", Text: " "},
	} {
		if request.Validate() == nil {
			t.Fatal("invalid request accepted")
		}
	}
}
