package main

import (
	"context"
	"errors"
	"testing"

	"expense-tracker/backend/services/monthlyreview"

	"github.com/stretchr/testify/require"
)

func TestPublisherRejectsUnsupportedActions(t *testing.T) {
	for _, action := range []string{"", "claim", "ack"} {
		t.Run(action, func(t *testing.T) {
			_, err := handle(t.Context(), publicationRequest{Action: action}, func(context.Context) (monthlyreview.PublicationResult, error) {
				t.Fatal("unexpected publication")
				return monthlyreview.PublicationResult{}, nil
			})
			require.ErrorContains(t, err, "unsupported")
		})
	}
}

func TestPublisherReturnsCountAndSanitizesErrors(t *testing.T) {
	request := publicationRequest{Action: "publish_monthly_reviews"}
	result, err := handle(t.Context(), request, func(ctx context.Context) (monthlyreview.PublicationResult, error) {
		_, bounded := ctx.Deadline()
		require.True(t, bounded)
		return monthlyreview.PublicationResult{Published: 2}, nil
	})
	require.NoError(t, err)
	require.Equal(t, 2, result.Published)

	_, err = handle(t.Context(), request, func(context.Context) (monthlyreview.PublicationResult, error) {
		return monthlyreview.PublicationResult{}, errors.New("private database detail")
	})
	require.EqualError(t, err, "monthly review publication failed")
}
