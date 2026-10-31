package llm

import (
	"testing"

	"expense-tracker/agent/internal/models"
)

func testPolicy() models.ModelPolicy {
	return models.ModelPolicy{Default: models.ModelProfile{Name: "configured-default", MaxOutputTokens: 1024},
		Router: models.ModelProfile{Name: "configured-router", MaxOutputTokens: 512}}
}

func testLimits() models.LLMLimits {
	return models.LLMLimits{MaxModelCalls: 6, MaxToolCalls: 2, MaxStateUpdates: 4,
		MaxContextBytes: 64 << 10, MaxResponseBytes: 32 << 10,
		MaxTotalResponseBytes: 256 << 10, MaxResponseEvents: 256}
}

func TestModelSelection(t *testing.T) {
	policy := testPolicy()
	for _, role := range []string{"", " ", "unknown", "configured-default", "default", "router"} {
		t.Run(role, func(t *testing.T) {
			selection, err := SelectModel(policy, role)
			if err != nil {
				t.Fatal(err)
			}
			expected := policy.Default
			if role == "router" {
				expected = policy.Router
			}
			known := role == "default" || role == "router"
			if selection.Selected != expected.Name || selection.MaxOutputTokens != expected.MaxOutputTokens || selection.Fallback == known || selection.Requested != role {
				t.Fatalf("unexpected model selection: %+v", selection)
			}
			if !known && selection.Reason == "" {
				t.Fatal("missing fallback reason")
			}
		})
	}
	for _, invalid := range []models.ModelPolicy{{}, {Default: policy.Default}, {Default: models.ModelProfile{Name: "test"}, Router: policy.Router}} {
		if _, err := SelectModel(invalid, "default"); err == nil {
			t.Fatal("invalid model policy accepted")
		}
	}
}

func TestSameModelKeepsRoleTokenLimits(t *testing.T) {
	policy := testPolicy()
	policy.Router.Name = policy.Default.Name
	for _, test := range []struct {
		role string
		want int32
	}{{"default", 1024}, {"router", 512}} {
		selection, err := SelectModel(policy, test.role)
		if err != nil {
			t.Fatal(err)
		}
		if selection.Selected != policy.Default.Name || selection.MaxOutputTokens != test.want {
			t.Fatal("model name replaced the configured role limit")
		}
	}
}
