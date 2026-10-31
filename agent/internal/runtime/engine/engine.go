// Package engine constructs ADK agents and runs individual ADK turns.
package engine

import (
	"cmp"
	"context"
	"fmt"
	"io/fs"
	"log/slog"
	"maps"
	"slices"

	"expense-tracker/agent/internal/llm"
	"expense-tracker/agent/internal/models"
	skillservice "expense-tracker/agent/internal/runtime/skill"
	"expense-tracker/agent/skills"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/tool"
)

// AppName scopes the shared ADK sessions and recorded conversations.
const AppName = "expense-agent"

// Dependencies supplies model, tool and ADK-session inputs for the engine.
type Dependencies struct {
	Models               llm.Factory
	Tools                map[string]tool.Tool
	CompletionChecks     map[string]func(agent.Context) error
	RouterTools          []string
	InstructionProviders map[string]llmagent.InstructionProvider
	AfterModelCallbacks  map[string]llmagent.AfterModelCallback
	Sessions             session.Service
	Skills               fs.FS
	ModelPolicy          models.ModelPolicy
	Limits               models.LLMLimits
	Logger               *slog.Logger
}

type Engine struct {
	runner        *runner.Runner
	agentMetadata map[string]models.SkillMetadata
	logger        *slog.Logger
}

// New builds one Chat coordinator with Task leaves from trusted bundled skills.
// Modes and tool implementations are runtime-owned, not skill-defined.
func New(ctx context.Context, deps Dependencies) (*Engine, error) {
	if deps.Models == nil {
		return nil, fmt.Errorf("explicit model factory required")
	}
	for name, granted := range deps.Tools {
		if granted == nil || granted.Name() != name {
			return nil, fmt.Errorf("invalid injected tool %q", name)
		}
	}
	if deps.Sessions == nil {
		deps.Sessions = session.InMemoryService()
	}
	if deps.Skills == nil {
		deps.Skills = skills.Files
	}
	if deps.Logger == nil {
		deps.Logger = slog.Default()
	}
	if err := deps.ModelPolicy.Validate(); err != nil {
		return nil, err
	}

	if err := deps.Limits.Validate(); err != nil {
		return nil, err
	}

	discovered, err := skillservice.Discover(ctx, deps.Skills)
	if err != nil {
		return nil, err
	}
	registry, err := skillservice.NewRegistry(discovered, slices.Collect(maps.Keys(deps.Tools)), deps.RouterTools)
	if err != nil {
		return nil, err
	}

	engine := &Engine{agentMetadata: map[string]models.SkillMetadata{}, logger: deps.Logger}

	var leaves []agent.Agent
	for _, skill := range registry.Leaves {
		leaf, err := engine.buildSkillAgent(ctx, deps, skill, nil)
		if err != nil {
			return nil, err
		}
		leaves = append(leaves, leaf)
	}
	root, err := engine.buildSkillAgent(ctx, deps, registry.Router, leaves)
	if err != nil {
		return nil, err
	}

	engine.runner, err = runner.New(runner.Config{
		AppName:           AppName,
		Agent:             root,
		SessionService:    deps.Sessions,
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, err
	}
	return engine, nil
}

func (engine *Engine) SkillMetadata() []models.SkillMetadata {
	return slices.SortedFunc(maps.Values(engine.agentMetadata), func(left, right models.SkillMetadata) int {
		return cmp.Compare(left.Name, right.Name)
	})
}
