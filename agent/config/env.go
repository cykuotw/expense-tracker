// Package config loads and validates Agent runtime configuration.
package config

import (
	"expense-tracker/agent/internal/models"
	"fmt"
	"os"
	"strconv"
	"strings"

	"expense-tracker/internal/localonly"
)

type Config struct {
	Mode             string
	ListenAddress    string
	WorkerURL        string
	StateDatabaseURL string
	GeminiAPIKey     string
	ModelPolicy      models.ModelPolicy
	Limits           models.LLMLimits
}

// Load reads exported environment variables and validates local runtime settings.
func Load() (Config, error) {
	if err := localonly.CheckRuntime(); err != nil {
		return Config{}, err
	}
	return loadFromLookup(os.LookupEnv)
}

func loadFromLookup(lookup func(string) (string, bool)) (Config, error) {
	cfg := Config{
		Mode:          "fixture",
		ListenAddress: "127.0.0.1:8090",
		ModelPolicy: models.ModelPolicy{
			Default: models.ModelProfile{Name: "gemini-3.8-flash", MaxOutputTokens: 2048},
			Router:  models.ModelProfile{Name: "gemini-3.5-flash-lite", MaxOutputTokens: 2048},
		},
		Limits: models.LLMLimits{MaxModelCalls: 6, MaxToolCalls: 2, MaxStateUpdates: 4,
			MaxContextBytes: 64 << 10, MaxResponseBytes: 32 << 10,
			MaxTotalResponseBytes: 256 << 10, MaxResponseEvents: 256},
	}
	setString := func(key string, target *string) {
		if value, ok := lookup(key); ok {
			*target = strings.TrimSpace(value)
		}
	}
	setString("AGENT_MODE", &cfg.Mode)
	setString("AGENT_LISTEN_ADDR", &cfg.ListenAddress)
	setString("AGENT_WORKER_URL", &cfg.WorkerURL)
	setString("GEMINI_API_KEY", &cfg.GeminiAPIKey)
	setString("AGENT_DEFAULT_MODEL", &cfg.ModelPolicy.Default.Name)
	setString("AGENT_ROUTER_MODEL", &cfg.ModelPolicy.Router.Name)
	for _, setting := range []struct {
		key    string
		target *int32
	}{
		{"AGENT_DEFAULT_MAX_OUTPUT_TOKENS", &cfg.ModelPolicy.Default.MaxOutputTokens},
		{"AGENT_ROUTER_MAX_OUTPUT_TOKENS", &cfg.ModelPolicy.Router.MaxOutputTokens},
		{"AGENT_MAX_MODEL_CALLS", &cfg.Limits.MaxModelCalls},
		{"AGENT_MAX_TOOL_CALLS", &cfg.Limits.MaxToolCalls},
		{"AGENT_MAX_STATE_UPDATES", &cfg.Limits.MaxStateUpdates},
		{"AGENT_MAX_CONTEXT_BYTES", &cfg.Limits.MaxContextBytes},
		{"AGENT_MAX_RESPONSE_BYTES", &cfg.Limits.MaxResponseBytes},
		{"AGENT_MAX_TOTAL_RESPONSE_BYTES", &cfg.Limits.MaxTotalResponseBytes},
		{"AGENT_MAX_RESPONSE_EVENTS", &cfg.Limits.MaxResponseEvents},
	} {
		if raw, ok := lookup(setting.key); ok {
			value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 32)
			if err != nil || value <= 0 {
				return Config{}, fmt.Errorf("%s must be a positive 32-bit integer", setting.key)
			}
			*setting.target = int32(value)
		}
	}
	for _, setting := range []struct{ key, value string }{
		{"AGENT_DEFAULT_MODEL", cfg.ModelPolicy.Default.Name},
		{"AGENT_ROUTER_MODEL", cfg.ModelPolicy.Router.Name},
	} {
		if setting.value == "" {
			return Config{}, fmt.Errorf("%s is required", setting.key)
		}
	}
	if err := cfg.Limits.Validate(); err != nil {
		return Config{}, fmt.Errorf("invalid Agent LLM limits: %w", err)
	}

	if value, ok := lookup("AGENT_STATE_DATABASE_URL"); ok {
		cfg.StateDatabaseURL = value
	}
	if cfg.Mode == "" {
		cfg.Mode = "fixture"
	}
	if cfg.ListenAddress == "" {
		cfg.ListenAddress = "127.0.0.1:8090"
	}
	if err := localonly.CheckAddress(cfg.ListenAddress); err != nil {
		return Config{}, fmt.Errorf("AGENT_LISTEN_ADDR requires a literal loopback address and port")
	}
	switch cfg.Mode {
	case "fixture":
	case "local":
		if strings.TrimSpace(cfg.StateDatabaseURL) == "" {
			return Config{}, fmt.Errorf("AGENT_STATE_DATABASE_URL is required for local persistence")
		}
		if cfg.WorkerURL == "" {
			return Config{}, fmt.Errorf("AGENT_WORKER_URL is required for local mode")
		}
	default:
		return Config{}, fmt.Errorf("unsupported AGENT_MODE")
	}
	if cfg.GeminiAPIKey == "" {
		return Config{}, fmt.Errorf("GEMINI_API_KEY is required for model inference")
	}
	return cfg, nil
}
