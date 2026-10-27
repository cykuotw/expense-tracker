package tracker

import (
	"testing"

	"expense-tracker/backend/config"
	"github.com/stretchr/testify/require"
)

func TestLocalAgentStartupGuards(t *testing.T) {
	cfg := config.Config{Mode: "debug", BackendURL: "127.0.0.1:8000"}
	require.NoError(t, ValidateLocalAgentMode(cfg))
	for _, address := range []string{"0.0.0.0:8000", ":8000", "192.0.2.1:8000", "localhost:8000"} {
		cfg.BackendURL = address
		require.Error(t, ValidateLocalAgentMode(cfg))
	}
	cfg.BackendURL = "127.0.0.1:8000"
	cfg.Mode = "release"
	require.Error(t, ValidateLocalAgentMode(cfg))
	cfg.Mode = "debug"
	t.Run("Lambda", func(t *testing.T) {
		t.Setenv("AWS_LAMBDA_RUNTIME_API", "synthetic")
		require.Error(t, ValidateLocalAgentMode(cfg))
	})
	original := config.BuildMode
	config.BuildMode = "release"
	t.Cleanup(func() { config.BuildMode = original })
	require.Error(t, ValidateLocalAgentMode(cfg))
}
