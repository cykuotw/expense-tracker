package getbalance

import (
	"expense-tracker/agent/internal/models"

	"google.golang.org/adk/v2/agent"
)

func balanceReadKey(ctx agent.Context) string {
	return "temp:balance_read:" + ctx.AgentName() + ":" + ctx.Path()
}

func recordBalanceRead(ctx agent.Context) error {
	return ctx.State().Set(balanceReadKey(ctx), ctx.InvocationID())
}

// RequireRead checks for a successful read in this task activation and invocation.
// Temporary state is excluded from durable session history.
func RequireRead(ctx agent.Context) error {
	invocation, err := ctx.State().Get(balanceReadKey(ctx))
	if err != nil || invocation != ctx.InvocationID() {
		return &models.Error{Code: models.Internal}
	}
	return nil
}
