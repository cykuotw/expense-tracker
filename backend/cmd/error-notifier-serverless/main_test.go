package main

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"testing"

	"expense-tracker/backend/internal/errornotifier"

	"github.com/stretchr/testify/require"
)

func resetHandlerCache() {
	handlerCache.Lock()
	defer handlerCache.Unlock()
	handlerCache.handler = nil
}

func TestRuntimeHandlerUsesWebhookEnvironmentAndCachesSuccess(t *testing.T) {
	resetHandlerCache()
	t.Cleanup(resetHandlerCache)
	t.Setenv("DISCORD_WEBHOOK_URL", "https://discord.com/api/webhooks/123456789012345678/abcdefghijklmnopqrstuvwxyz_ABCD")
	t.Setenv("DEPLOYMENT_ENVIRONMENT", "production")
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "error-notifier")

	first, err := runtimeHandler()
	require.NoError(t, err)
	second, err := runtimeHandler()
	require.NoError(t, err)
	require.Same(t, first, second)
}

func TestRuntimeHandlerRejectsInvalidWebhookWithoutExposingIt(t *testing.T) {
	resetHandlerCache()
	t.Cleanup(resetHandlerCache)
	t.Setenv("DISCORD_WEBHOOK_URL", "https://example.com/CANARY_WEBHOOK_SECRET")
	t.Setenv("DEPLOYMENT_ENVIRONMENT", "production")
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "error-notifier")

	_, err := runtimeHandler()
	require.Error(t, err)
	require.False(t, errornotifier.IsRetryable(err))
	require.NotContains(t, err.Error(), "CANARY_WEBHOOK_SECRET")
}

func TestDatabaseEventCannotFallBackToApplicationWebhook(t *testing.T) {
	t.Setenv("DISCORD_WEBHOOK_URL", "https://discord.com/api/webhooks/123456789012345678/abcdefghijklmnopqrstuvwxyz_ABCD")
	t.Setenv("DATABASE_DISCORD_WEBHOOK_URL", "https://invalid.example/PRIVATE_DATABASE_HOOK")
	err := handle(t.Context(), []byte(`{"Records":[{"eventSource":"aws:s3"}]}`))
	require.Error(t, err)
	require.False(t, errornotifier.IsRetryable(err))
	require.NotContains(t, err.Error(), "PRIVATE_DATABASE_HOOK")
}

func TestLegacyCloudWatchControlMessageStillWorks(t *testing.T) {
	resetHandlerCache()
	t.Cleanup(resetHandlerCache)
	t.Setenv("DISCORD_WEBHOOK_URL", "https://discord.com/api/webhooks/123456789012345678/abcdefghijklmnopqrstuvwxyz_ABCD")
	t.Setenv("DEPLOYMENT_ENVIRONMENT", "production")
	t.Setenv("AWS_LAMBDA_FUNCTION_NAME", "error-notifier")
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err := writer.Write([]byte(`{"messageType":"CONTROL_MESSAGE"}`))
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	raw := []byte(`{"awslogs":{"data":"` + base64.StdEncoding.EncodeToString(compressed.Bytes()) + `"}}`)
	require.NoError(t, handle(t.Context(), raw))
}
