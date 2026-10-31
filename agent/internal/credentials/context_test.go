package credentials

import (
	"context"
	"slices"
	"testing"
)

func TestCredentialsStayRequestScoped(t *testing.T) {
	parent, cancel := context.WithCancel(t.Context())
	ctx := WithAccessToken(parent, "opaque-access")
	ctx = WithSecrets(ctx, "provider-key")
	if AccessToken(parent) != "" || len(Secrets(parent)) != 0 {
		t.Fatal("credentials escaped into the parent context")
	}
	if AccessToken(ctx) != "opaque-access" || !slices.Equal(Secrets(ctx), []string{"opaque-access", "provider-key"}) {
		t.Fatal("request credentials were not retained for transport and filtering")
	}
	copy := Secrets(ctx)
	copy[0] = "modified"
	if Secrets(ctx)[0] != "opaque-access" {
		t.Fatal("a filter can mutate another consumer's secret list")
	}
	cancel()
	if ctx.Err() != context.Canceled {
		t.Fatal("credential context did not preserve cancellation")
	}
}
