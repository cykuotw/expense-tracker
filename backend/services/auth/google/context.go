package google

import (
	"context"
	"expense-tracker/backend/types"
)

type verifiedClaimsContextKey struct{}

func ContextWithVerifiedClaims(ctx context.Context, claims *types.VerifiedGoogleClaims) context.Context {
	return context.WithValue(ctx, verifiedClaimsContextKey{}, claims)
}

func VerifiedClaimsFromContext(ctx context.Context) (*types.VerifiedGoogleClaims, error) {
	if ctx == nil {
		return nil, types.ErrGoogleClaimsUnavailable
	}

	claims, ok := ctx.Value(verifiedClaimsContextKey{}).(*types.VerifiedGoogleClaims)
	if !ok || claims == nil {
		return nil, types.ErrGoogleClaimsUnavailable
	}

	return claims, nil
}
