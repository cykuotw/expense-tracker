package fixtures

import (
	"context"

	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/tools/functiontool/getbalance"
)

type BalanceClient struct{}

func (BalanceClient) GetBalance(_ context.Context, group string) (getbalance.Result, error) {
	if group != "demo" {
		return getbalance.Result{}, &models.Error{Code: models.NotFound}
	}
	return getbalance.Result{
		GroupID: group,
		Balances: []getbalance.Balance{
			{
				From:     "Alice",
				To:       "Bob",
				Amount:   "12.34",
				Currency: "CAD",
			},
		},
	}, nil
}
