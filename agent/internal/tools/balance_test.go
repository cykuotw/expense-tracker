package tools

import (
	"context"
	"errors"
	"testing"

	"expense-tracker/agent/internal/models"
)

type clientFunc func(context.Context, string) (BalanceResult, error)

func (client clientFunc) GetBalance(ctx context.Context, groupID string) (BalanceResult, error) {
	return client(ctx, groupID)
}

func TestBalanceBoundary(t *testing.T) {
	ctx := WithGroup(models.WithBudget(t.Context()), "demo")
	result, err := Fetch(ctx, FixtureBalanceClient{}, BalanceInput{GroupID: "demo"})
	if err != nil {
		t.Fatal(err)
	}
	if result.Balances[0].Amount != "12.34" || result.Balances[0].Currency != "CAD" || !Succeeded(ctx) {
		t.Fatal(result)
	}

	called := false
	unexpectedClient := clientFunc(func(context.Context, string) (BalanceResult, error) {
		called = true
		return BalanceResult{}, nil
	})
	_, err = Fetch(ctx, unexpectedClient, BalanceInput{GroupID: "other"})
	if PublicCode(err) != Forbidden || called {
		t.Fatalf("group escape: called=%v err=%v", called, err)
	}

	_, err = Fetch(models.WithBudget(t.Context()), unexpectedClient, BalanceInput{GroupID: "demo"})
	if PublicCode(err) != Forbidden || called {
		t.Fatal("missing scope accepted")
	}
	for range 2 {
		_, err = Fetch(ctx, FixtureBalanceClient{}, BalanceInput{GroupID: "demo"})
	}
	if !errors.Is(err, models.ErrBudget) {
		t.Fatalf("tool budget not enforced: %v", err)
	}
}

func TestBalanceErrorsAndMalformedResults(t *testing.T) {
	for _, code := range []Code{
		Validation, Unauthenticated, Forbidden, NotFound, Unavailable, Internal,
	} {
		ctx := WithGroup(models.WithBudget(t.Context()), "demo")
		failingClient := clientFunc(func(context.Context, string) (BalanceResult, error) {
			return BalanceResult{}, &Error{Code: code}
		})
		_, err := Fetch(ctx, failingClient, BalanceInput{GroupID: "demo"})
		if PublicCode(err) != code {
			t.Fatalf("lost code %s: %v", code, err)
		}
	}

	ctx := WithGroup(models.WithBudget(t.Context()), "demo")
	failingClient := clientFunc(func(context.Context, string) (BalanceResult, error) {
		return BalanceResult{}, errors.New("private credential-bearing transport error")
	})
	_, err := Fetch(ctx, failingClient, BalanceInput{GroupID: "demo"})
	if err.Error() != "unavailable" {
		t.Fatal("raw error escaped", err)
	}

	results := []BalanceResult{
		{GroupID: "other"},
		{
			GroupID: "demo",
			Balances: []Balance{
				{Amount: "NaN", Currency: "CAD"},
			},
		},
		{
			GroupID: "demo",
			Balances: []Balance{
				{Amount: "12.34", Currency: "cad"},
			},
		},
	}
	for _, result := range results {
		ctx := WithGroup(models.WithBudget(t.Context()), "demo")
		malformedClient := clientFunc(func(context.Context, string) (BalanceResult, error) {
			return result, nil
		})
		_, err := Fetch(ctx, malformedClient, BalanceInput{GroupID: "demo"})
		if PublicCode(err) != Internal || Succeeded(ctx) {
			t.Fatal("malformed result accepted", err)
		}
	}
}
