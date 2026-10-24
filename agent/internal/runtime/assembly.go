package runtime

import (
	"context"
	"fmt"
	"log/slog"
	"slices"

	"expense-tracker/agent/internal/models"
	skillservice "expense-tracker/agent/internal/service/skill"
	"expense-tracker/agent/internal/tools"
	"expense-tracker/agent/skills"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/tool"
)

// New builds one Chat coordinator with Task leaves from trusted bundled skills.
// Modes and tool implementations are runtime-owned, not skill-defined.
func New(ctx context.Context, config Config) (*App, error) {
	if config.Models == nil || config.Balance == nil {
		return nil, fmt.Errorf("explicit model factory and balance client required")
	}
	if config.Sessions == nil {
		config.Sessions = session.InMemoryService()
	}
	if config.Skills == nil {
		config.Skills = skills.Files
	}
	if config.DefaultModel == "" {
		config.DefaultModel = models.DefaultModel
	}
	if config.Logger == nil {
		config.Logger = slog.Default()
	}
	if _, err := models.Resolve("", config.DefaultModel); err != nil {
		return nil, err
	}

	discovered, err := skillservice.Discover(ctx, config.Skills)
	if err != nil {
		return nil, err
	}
	registry, err := skillservice.NewRegistry(discovered, tools.Names())
	if err != nil {
		return nil, err
	}

	app := &App{
		diagnostics:   map[string]Diagnostic{},
		conversations: map[string]*conversation{},
		agentSkills:   map[string]string{},
		logger:        config.Logger,
	}
	var leaves []agent.Agent
	for _, skill := range registry.Leaves {
		leaf, err := app.buildAgent(ctx, config, skill, nil)
		if err != nil {
			return nil, err
		}
		leaves = append(leaves, leaf)
	}
	root, err := app.buildAgent(ctx, config, registry.Router, leaves)
	if err != nil {
		return nil, err
	}

	app.runner, err = runner.New(runner.Config{
		AppName:           sessionAppName,
		Agent:             root,
		SessionService:    config.Sessions,
		AutoCreateSession: true,
	})
	if err != nil {
		return nil, err
	}
	return app, nil
}

func (app *App) buildAgent(
	ctx context.Context,
	config Config,
	skill skillservice.Skill,
	children []agent.Agent,
) (agent.Agent, error) {
	resolution, err := models.Resolve(skill.ModelPreference, config.DefaultModel)
	if err != nil {
		return nil, err
	}
	provider, err := config.Models(ctx, skill.Name, resolution.Resolved)
	if err != nil {
		return nil, fmt.Errorf("construct %s model: %w", skill.Name, err)
	}
	if provider == nil {
		return nil, fmt.Errorf("nil model for %s", skill.Name)
	}

	var grantedTools []tool.Tool
	for _, name := range skill.Tools {
		granted, err := tools.Build(name, config.Balance)
		if err != nil {
			return nil, err
		}
		grantedTools = append(grantedTools, granted)
	}

	mode := llmagent.ModeTask
	if skill.Name == "orchestrator" {
		mode = llmagent.ModeChat
	}
	agentName := skillservice.AgentName(skill.Name)
	agentConfig := llmagent.Config{
		Name:                     agentName,
		Description:              skill.Description,
		Model:                    models.Limited(provider),
		Mode:                     mode,
		IncludeContents:          llmagent.IncludeContentsDefault,
		Tools:                    grantedTools,
		SubAgents:                children,
		DisallowTransferToParent: true,
		DisallowTransferToPeers:  true,
		InstructionProvider: func(agent.ReadonlyContext) (string, error) {
			return skill.Instructions, nil
		},
		BeforeToolCallbacks: []llmagent.BeforeToolCallback{
			validateToolCall(skill, children),
		},
	}
	configuredAgent, err := llmagent.New(agentConfig)
	if err != nil {
		return nil, err
	}

	app.agentSkills[agentName] = skill.Name
	app.diagnostics[skill.Name] = Diagnostic{
		Skill:        skill.Name,
		Model:        resolution,
		SkillVersion: skill.Version,
	}
	return configuredAgent, nil
}

// Validate even framework-managed finish/delegation tools. Skill text and model
// output cannot expand grants or finish a balance task without a successful read.
func validateToolCall(skill skillservice.Skill, children []agent.Agent) llmagent.BeforeToolCallback {
	return func(ctx agent.Context, called tool.Tool, _ map[string]any) (map[string]any, error) {
		if skill.Name == "orchestrator" {
			for _, child := range children {
				if called.Name() == child.Name() {
					return nil, nil
				}
			}
			return nil, &tools.Error{Code: tools.Forbidden}
		}

		if slices.Contains(skill.Tools, called.Name()) {
			return nil, nil
		}
		if called.Name() != "finish_task" {
			return nil, &tools.Error{Code: tools.Forbidden}
		}

		requiresBalanceRead := slices.Contains(skill.Tools, "get_balance")
		if requiresBalanceRead && !tools.ReadSucceeded(ctx) {
			return nil, &tools.Error{Code: tools.Internal}
		}
		return nil, nil
	}
}
