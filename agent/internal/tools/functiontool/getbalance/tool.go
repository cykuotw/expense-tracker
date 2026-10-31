package getbalance

import (
	"context"
	"regexp"
	"sync/atomic"
	"time"
	"uuid"

	"expense-tracker/agent/internal/llm"
	"expense-tracker/agent/internal/models"
	sessionstore "expense-tracker/agent/internal/store/session"
	"expense-tracker/internal/agentcontract"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/functiontool"
)

const ToolName = "get_balance"

type Input struct {
	GroupID string `json:"groupId"`
}

type Balance = agentcontract.Balance
type Result = agentcontract.BalanceResult

// The implementation captures trusted actor credentials out of band. Models
// receive resource selectors only. No database or accounting logic belongs here.
type Client interface {
	GetBalance(ctx context.Context, groupID string) (Result, error)
}

type groupKey struct{}

type groupScope struct {
	groupID string
	success atomic.Bool
}

func WithGroup(ctx context.Context, group string) context.Context {
	return context.WithValue(ctx, groupKey{}, &groupScope{groupID: group})
}

func Succeeded(ctx context.Context) bool {
	scope, ok := ctx.Value(groupKey{}).(*groupScope)
	return ok && scope.success.Load()
}

const (
	maxBalanceRows      = agentcontract.MaxBalanceRows
	maxParticipantBytes = 128
)

var amountPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,15})(\.[0-9]{1,8})?$`)
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
var groupIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

func Fetch(ctx context.Context, client Client, input Input) (Result, error) {
	scope, ok := ctx.Value(groupKey{}).(*groupScope)
	if !ok || scope.groupID == "" || input.GroupID != scope.groupID {
		return Result{}, &models.Error{Code: models.Forbidden}
	}
	if err := llm.ConsumeToolCall(ctx); err != nil {
		return Result{}, err
	}
	callID := uuid.New().String()
	ref := sessionstore.CurrentCall(ctx)
	if ref.CallID != "" {
		callID = ref.CallID
	}
	started := time.Now()
	if err := sessionstore.RecordDiagnostic(ctx, "tool.started", "", map[string]any{"callId": callID, "name": "get_balance", "input": input, "agent": ref}); err != nil {
		return Result{}, err
	}
	result, err := client.GetBalance(ctx, scope.groupID)
	if err != nil {
		if storageErr := sessionstore.RecordDiagnostic(ctx, "tool.finished", "", map[string]any{"callId": callID, "error": models.PublicCode(err), "durationMs": time.Since(started).Milliseconds()}); storageErr != nil {
			return Result{}, storageErr
		}
		return Result{}, &models.Error{Code: models.PublicCode(err)}
	}
	if result.GroupID != scope.groupID || len(result.Balances) > maxBalanceRows {
		return Result{}, &models.Error{Code: models.Internal}
	}
	for _, balance := range result.Balances {
		invalidBalance := len(balance.From) > maxParticipantBytes ||
			len(balance.To) > maxParticipantBytes ||
			!amountPattern.MatchString(balance.Amount) ||
			!currencyPattern.MatchString(balance.Currency)
		if invalidBalance {
			return Result{}, &models.Error{Code: models.Internal}
		}
	}
	if err := sessionstore.RecordDiagnostic(ctx, "tool.finished", "", map[string]any{"callId": callID, "result": result, "durationMs": time.Since(started).Milliseconds()}); err != nil {
		return Result{}, err
	}
	scope.success.Store(true)
	return result, nil
}

func NewTool(client Client) (tool.Tool, error) {
	config := functiontool.Config{
		Name: ToolName,
		Description: "Read authoritative balances for the supplied group, " +
			"preserving exact amount strings and original currencies.",
	}
	return functiontool.New(
		config,
		func(ctx agent.Context, input Input) (Result, error) {
			if !groupIDPattern.MatchString(input.GroupID) {
				return Result{}, &models.Error{Code: models.Validation}
			}
			// The task chooses a resource selector; the injected backend client
			// must authorize it for the trusted actor on every call.
			tracked := sessionstore.WithCall(ctx, models.CallRef{InvocationID: ctx.InvocationID(), CallID: ctx.FunctionCallID(), Agent: ctx.AgentName(), Branch: ctx.Branch(), Path: ctx.Path()})
			result, err := Fetch(WithGroup(tracked, input.GroupID), client, input)
			if err == nil {
				err = recordBalanceRead(ctx)
			}
			return result, err
		},
	)
}
