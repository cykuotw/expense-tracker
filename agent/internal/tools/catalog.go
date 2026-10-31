package tools

import (
	"expense-tracker/agent/internal/tools/functiontool/getbalance"
	"expense-tracker/agent/internal/tools/functiontool/updatestate"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/tool"
)

// NewCatalog constructs application tools outside the runtime coordinator.
func NewCatalog(client getbalance.Client) (map[string]tool.Tool, error) {
	balance, err := getbalance.NewTool(client)
	if err != nil {
		return nil, err
	}
	state, err := updatestate.NewTool()
	if err != nil {
		return nil, err
	}
	return map[string]tool.Tool{getbalance.ToolName: balance, updatestate.ToolName: state}, nil
}

func RouterTools() []string { return []string{updatestate.ToolName} }

func InstructionProviders() map[string]llmagent.InstructionProvider {
	return map[string]llmagent.InstructionProvider{updatestate.ToolName: updatestate.Instructions}
}

func AfterModelCallbacks() map[string]llmagent.AfterModelCallback {
	return map[string]llmagent.AfterModelCallback{updatestate.ToolName: updatestate.ValidateModelResponse}
}

func CompletionChecks() map[string]func(agent.Context) error {
	return map[string]func(agent.Context) error{getbalance.ToolName: getbalance.RequireRead}
}
