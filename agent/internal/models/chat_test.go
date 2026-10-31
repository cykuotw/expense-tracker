package models

import (
	"strings"
	"testing"
)

func TestChatRequestValidation(t *testing.T) {
	for _, test := range []struct {
		name    string
		request ChatRequest
		valid   bool
	}{
		{"valid", ChatRequest{ConversationID: "chat", MessageID: "one", Text: "hello"}, true},
		{"optional run", ChatRequest{ConversationID: "chat", RunID: "run-1", MessageID: "one", Text: "hello"}, true},
		{"text limit", ChatRequest{ConversationID: "chat", MessageID: "one", Text: strings.Repeat("x", MaxTextBytes)}, true},
		{"invalid conversation", ChatRequest{ConversationID: "../chat", MessageID: "one", Text: "hello"}, false},
		{"missing message", ChatRequest{ConversationID: "chat", Text: "hello"}, false},
		{"invalid run", ChatRequest{ConversationID: "chat", RunID: "run/1", MessageID: "one", Text: "hello"}, false},
		{"blank text", ChatRequest{ConversationID: "chat", MessageID: "one", Text: " \n"}, false},
		{"invalid UTF8", ChatRequest{ConversationID: "chat", MessageID: "one", Text: string([]byte{255})}, false},
		{"text over limit", ChatRequest{ConversationID: "chat", MessageID: "one", Text: strings.Repeat("x", MaxTextBytes+1)}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := test.request.Validate()
			if test.valid {
				if err != nil {
					t.Fatal(err)
				}
			} else if err == nil || PublicCode(err) != Validation {
				t.Fatalf("expected validation failure, got %v", err)
			}
		})
	}
}
