package integration_test

import (
	"strings"
	"testing"
	"testing/fstest"

	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/runtime"
)

func TestRuntimeWithoutDomainTools(t *testing.T) {
	files := fstest.MapFS{
		"orchestrator/SKILL.md": {Data: []byte("---\nname: orchestrator\ndescription: Coordinate tasks.\n---\nDelegate requests.\n")},
		"echo/SKILL.md":         {Data: []byte("---\nname: echo\ndescription: Echo text.\n---\nEcho the request.\n")},
	}
	app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(), Models: fixtures.ModelFactory, Skills: files})
	if err != nil {
		t.Fatal(err)
	}
	_, answer, err := runMessage(t, app, "echo-thread", "one", "echo hello")
	if err != nil || !strings.Contains(answer, "echo hello") {
		t.Fatalf("runtime without domain dependencies: answer=%q err=%v", answer, err)
	}
}
