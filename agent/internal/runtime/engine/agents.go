package engine

import (
	"context"
	"fmt"
	"slices"

	"expense-tracker/agent/internal/llm"
	"expense-tracker/agent/internal/models"
	skillservice "expense-tracker/agent/internal/runtime/skill"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/tool"
)

func (engine *Engine) buildSkillAgent(
	ctx context.Context,
	deps Dependencies,
	skill skillservice.Skill,
	children []agent.Agent,
) (agent.Agent, error) {
	selection, err := llm.SelectModel(deps.ModelPolicy, skill.ModelRole)
	if err != nil {
		return nil, err
	}
	provider, err := deps.Models(ctx, skill.Name, selection.Selected)
	if err != nil {
		return nil, fmt.Errorf("construct %s model: %w", skill.Name, err)
	}
	if provider == nil {
		return nil, fmt.Errorf("nil model for %s", skill.Name)
	}

	var grantedTools []tool.Tool
	var afterModelCallbacks []llmagent.AfterModelCallback
	for _, name := range skill.Tools {
		grantedTools = append(grantedTools, deps.Tools[name])
		if callback := deps.AfterModelCallbacks[name]; callback != nil {
			afterModelCallbacks = append(afterModelCallbacks, callback)
		}
	}
	mode := llmagent.ModeTask
	if skill.Name == "orchestrator" {
		mode = llmagent.ModeChat
	}
	agentName := skillservice.AgentName(skill.Name)
	agentConfig := llmagent.Config{
		Name:                     agentName,
		Description:              skill.Description,
		Model:                    llm.WithLimitsAndRecording(provider, deps.Limits, selection.MaxOutputTokens, skill.Name, skill.Version),
		Mode:                     mode,
		IncludeContents:          llmagent.IncludeContentsDefault,
		Tools:                    grantedTools,
		SubAgents:                children,
		DisallowTransferToParent: true,
		DisallowTransferToPeers:  true,
		InstructionProvider: func(ctx agent.ReadonlyContext) (string, error) {
			instructions := skill.Instructions
			for _, name := range skill.Tools {
				if provider := deps.InstructionProviders[name]; provider != nil {
					extra, err := provider(ctx)
					if err != nil {
						return "", err
					}
					instructions += extra
				}
			}
			return instructions, nil
		},
		BeforeToolCallbacks: []llmagent.BeforeToolCallback{
			newToolCallValidator(skill, children, deps.CompletionChecks),
		},
		AfterModelCallbacks: afterModelCallbacks,
	}
	configuredAgent, err := llmagent.New(agentConfig)
	if err != nil {
		return nil, err
	}

	engine.agentMetadata[agentName] = models.SkillMetadata{
		Name:           skill.Name,
		Version:        skill.Version,
		ModelSelection: selection,
	}
	return configuredAgent, nil
}

// Validate even framework-managed finish/delegation tools. Skill text and model
// output cannot expand grants. Tool-specific completion checks are injected.
func newToolCallValidator(skill skillservice.Skill, children []agent.Agent, completionChecks map[string]func(agent.Context) error) llmagent.BeforeToolCallback {
	return func(ctx agent.Context, called tool.Tool, _ map[string]any) (map[string]any, error) {
		if slices.Contains(skill.Tools, called.Name()) {
			return nil, nil
		}
		if skill.Name == "orchestrator" {
			for _, child := range children {
				if called.Name() == child.Name() {
					return nil, nil
				}
			}
			return nil, &models.Error{Code: models.Forbidden}
		}

		if called.Name() != "finish_task" {
			return nil, &models.Error{Code: models.Forbidden}
		}

		for _, name := range skill.Tools {
			if check := completionChecks[name]; check != nil {
				if err := check(ctx); err != nil {
					return nil, err
				}
			}
		}
		return nil, nil
	}
}
