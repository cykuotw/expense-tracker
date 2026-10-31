package getbalance_test

import (
	"context"
	"errors"
	"testing"

	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/llm"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/tools/functiontool/getbalance"
)

type clientFunc func(context.Context, string) (getbalance.Result, error)

func (client clientFunc) GetBalance(ctx context.Context, groupID string) (getbalance.Result, error) {
	return client(ctx, groupID)
}

func TestBalanceBoundary(t *testing.T) {
	ctx := getbalance.WithGroup(llm.WithRunBudget(t.Context(), fixtures.Limits()), "demo")
	result, err := getbalance.Fetch(ctx, fixtures.BalanceClient{}, getbalance.Input{GroupID: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Balances[0].Amount != "12.34" || result.Balances[0].Currency != "CAD" || !getbalance.Succeeded(ctx) {
		t.Fatal(result)
	}

	called := false
	unexpectedClient := clientFunc(func(context.Context, string) (getbalance.Result, error) {
		called = true
		return getbalance.Result{}, nil
	})
	_, err = getbalance.Fetch(ctx, unexpectedClient, getbalance.Input{GroupID: "other"})
	if models.PublicCode(err) != models.Forbidden || called {
		t.Fatalf("group escape: called=%v err=%v", called, err)
	}

	_, err = getbalance.Fetch(llm.WithRunBudget(t.Context(), fixtures.Limits()), unexpectedClient, getbalance.Input{GroupID: "demo"})
	if models.PublicCode(err) != models.Forbidden || called {
		t.Fatal("missing scope accepted")
	}
	for range 2 {
		_, err = getbalance.Fetch(ctx, fixtures.BalanceClient{}, getbalance.Input{GroupID: "demo"})
	}
	if !errors.Is(err, models.ErrBudget) {
		t.Fatalf("tool budget not enforced: %v", err)
	}
}

func TestBalanceErrorsAndMalformedResults(t *testing.T) {
	for _, code := range []models.Code{
		models.Validation, models.Unauthenticated, models.Forbidden, models.NotFound, models.Unavailable, models.Internal,
	} {
		ctx := getbalance.WithGroup(llm.WithRunBudget(t.Context(), fixtures.Limits()), "demo")
		failingClient := clientFunc(func(context.Context, string) (getbalance.Result, error) {
			return getbalance.Result{}, &models.Error{Code: code}
		})
		_, err := getbalance.Fetch(ctx, failingClient, getbalance.Input{GroupID: "demo"})
		if models.PublicCode(err) != code {
			t.Fatalf("lost code %s: %v", code, err)
		}
	}

	ctx := getbalance.WithGroup(llm.WithRunBudget(t.Context(), fixtures.Limits()), "demo")
	failingClient := clientFunc(func(context.Context, string) (getbalance.Result, error) {
		return getbalance.Result{}, errors.New("private credential-bearing transport error")
	})
	_, err := getbalance.Fetch(ctx, failingClient, getbalance.Input{GroupID: "demo"})
	if err.Error() != "unavailable" {
		t.Fatal("raw error escaped", err)
	}

	results := []getbalance.Result{
		{GroupID: "other"},
		{
			GroupID: "demo",
			Balances: []getbalance.Balance{
				{Amount: "NaN", Currency: "CAD"},
			},
		},
		{
			GroupID: "demo",
			Balances: []getbalance.Balance{
				{Amount: "12.34", Currency: "cad"},
			},
		},
	}
	for _, result := range results {
		ctx := getbalance.WithGroup(llm.WithRunBudget(t.Context(), fixtures.Limits()), "demo")
		malformedClient := clientFunc(func(context.Context, string) (getbalance.Result, error) {
			return result, nil
		})
		_, err := getbalance.Fetch(ctx, malformedClient, getbalance.Input{GroupID: "demo"})
		if models.PublicCode(err) != models.Internal || getbalance.Succeeded(ctx) {
			t.Fatal("malformed result accepted", err)
		}
	}
}
