package llm

import (
	"testing"

	"google.golang.org/genai"
)

func TestGeminiConstructionWithoutInference(t *testing.T) {
	factory := NewGeminiFactory(genai.ClientConfig{
		APIKey:  "synthetic-not-a-real-key",
		Backend: genai.BackendGeminiAPI,
	})
	for _, name := range []string{"configured-default", "configured-router", "another-operator-model"} {
		provider, err := factory(t.Context(), "get-balance", name)
		if err != nil {
			t.Fatal(err)
		}
		if provider.Name() != name {
			t.Fatal(provider.Name())
		}
	}
	if _, err := factory(t.Context(), "get-balance", " "); err == nil {
		t.Fatal("blank model accepted")
	}
}
