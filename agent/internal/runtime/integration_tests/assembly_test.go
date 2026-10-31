package integration_test

import (
	"context"
	"testing"
	"time"

	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/runtime"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
)

func TestDurableAssemblyRejectsIncompleteDependenciesBeforeBuildingModels(t *testing.T) {
	for _, test := range []struct {
		name         string
		sessions     bool
		authenticate bool
		want         string
	}{
		{"missing sessions", false, true, "explicit persistent sessions"},
		{"missing authentication", true, false, "explicit authentication"},
	} {
		t.Run(test.name, func(t *testing.T) {
			constructions := 0
			deps := runtime.Dependencies{
				ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(), Records: &recordedRepository{},
				Models: func(ctx context.Context, skill, name string) (model.LLM, error) {
					constructions++
					return fixtures.ModelFactory(ctx, skill, name)
				},
			}
			if test.sessions {
				deps.Sessions = session.InMemoryService()
			}
			if test.authenticate {
				deps.Authenticate = func(context.Context) (models.Identity, error) {
					return models.Identity{ID: "owner", ExpiresAt: time.Now().Add(time.Hour)}, nil
				}
			}
			_, err := runtime.New(t.Context(), deps)
			require.ErrorContains(t, err, test.want)
			require.Zero(t, constructions, "durable wiring must not silently select an in-memory session fallback")
		})
	}
}
