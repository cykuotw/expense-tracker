// Package credentials carries request credentials outside serialized model payloads.
package credentials

import (
	"context"
	"slices"
)

type tokenKey struct{}
type secretsKey struct{}

func WithAccessToken(ctx context.Context, token string) context.Context {
	return context.WithValue(WithSecrets(ctx, token), tokenKey{}, token)
}

func AccessToken(ctx context.Context) string {
	token, _ := ctx.Value(tokenKey{}).(string)
	return token
}

func WithSecrets(ctx context.Context, secrets ...string) context.Context {
	old, _ := ctx.Value(secretsKey{}).([]string)
	return context.WithValue(ctx, secretsKey{}, append(slices.Clone(old), secrets...))
}

// Secrets returns a copy for credential filtering; callers must never log these values.
func Secrets(ctx context.Context) []string {
	values, _ := ctx.Value(secretsKey{}).([]string)
	return slices.Clone(values)
}
