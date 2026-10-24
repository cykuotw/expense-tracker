package runtime

import (
	"bytes"
	"encoding/json/v2"
	"log/slog"
	"strings"
	"testing"

	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/tools"
)

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
	app, err := New(t.Context(), Config{
		Models:  models.FixtureFactory,
		Balance: tools.FixtureBalanceClient{},
		Logger:  slog.New(slog.NewJSONHandler(&buffer, nil)),
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
	app, err := New(t.Context(), Config{
		Models:  models.FixtureFactory,
		Balance: unavailableBalance{},
		Logger:  slog.New(slog.NewJSONHandler(&buffer, nil)),
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
