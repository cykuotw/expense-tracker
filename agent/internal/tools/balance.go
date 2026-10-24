package tools

import (
	"context"
	"regexp"
	"sync/atomic"

	"expense-tracker/agent/internal/models"
	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/tool"
	"google.golang.org/adk/v2/tool/functiontool"
)

type BalanceInput struct {
	GroupID string `json:"groupId"`
}

type Balance struct {
	From     string `json:"from"`
	To       string `json:"to"`
	Amount   string `json:"amount"`
	Currency string `json:"currency"`
}

type BalanceResult struct {
	GroupID  string    `json:"groupId"`
	Balances []Balance `json:"balances"`
}

// The implementation captures trusted actor credentials out of band. Models
// receive resource selectors only. No database or accounting logic belongs here.
type BalanceClient interface {
	GetBalance(ctx context.Context, groupID string) (BalanceResult, error)
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
	maxBalanceRows      = 100
	maxParticipantBytes = 128
)

var amountPattern = regexp.MustCompile(`^(0|[1-9][0-9]{0,15})(\.[0-9]{1,8})?$`)
var currencyPattern = regexp.MustCompile(`^[A-Z]{3}$`)
var groupIDPattern = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

func Fetch(ctx context.Context, client BalanceClient, input BalanceInput) (BalanceResult, error) {
	scope, ok := ctx.Value(groupKey{}).(*groupScope)
	if !ok || scope.groupID == "" || input.GroupID != scope.groupID {
		return BalanceResult{}, &Error{Code: Forbidden}
	}
	if err := models.TakeTool(ctx); err != nil {
		return BalanceResult{}, err
	}
	result, err := client.GetBalance(ctx, scope.groupID)
	if err != nil {
		return BalanceResult{}, &Error{Code: PublicCode(err)}
	}
	if result.GroupID != scope.groupID || len(result.Balances) > maxBalanceRows {
		return BalanceResult{}, &Error{Code: Internal}
	}
	for _, balance := range result.Balances {
		invalidBalance := len(balance.From) > maxParticipantBytes ||
			len(balance.To) > maxParticipantBytes ||
			!amountPattern.MatchString(balance.Amount) ||
			!currencyPattern.MatchString(balance.Currency)
		if invalidBalance {
			return BalanceResult{}, &Error{Code: Internal}
		}
	}
	scope.success.Store(true)
	return result, nil
}

func NewBalance(client BalanceClient) (tool.Tool, error) {
	config := functiontool.Config{
		Name: "get_balance",
		Description: "Read authoritative balances for the supplied group, " +
			"preserving exact amount strings and original currencies.",
	}
	return functiontool.New(
		config,
		func(ctx agent.Context, input BalanceInput) (BalanceResult, error) {
			if !groupIDPattern.MatchString(input.GroupID) {
				return BalanceResult{}, &Error{Code: Validation}
			}
			// The task chooses a resource selector; the injected backend client
			// must authorize it for the trusted actor on every call.
			result, err := Fetch(WithGroup(ctx, input.GroupID), client, input)
			if err == nil {
				recordRead(ctx)
			}
			return result, err
		},
	)
}

type FixtureBalanceClient struct{}

func (FixtureBalanceClient) GetBalance(_ context.Context, group string) (BalanceResult, error) {
	if group != "demo" {
		return BalanceResult{}, &Error{Code: NotFound}
	}
	return BalanceResult{
		GroupID: group,
		Balances: []Balance{
			{
				From:     "Alice",
				To:       "Bob",
				Amount:   "12.34",
				Currency: "CAD",
			},
		},
	}, nil
}
