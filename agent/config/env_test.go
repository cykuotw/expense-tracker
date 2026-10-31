package config

import (
	"strings"
	"testing"

	"expense-tracker/agent/internal/models"
)

func TestListenAddressConfig(t *testing.T) {
	for _, test := range []struct {
		name    string
		address string
		want    string
	}{
		{name: "default", want: "127.0.0.1:8090"},
		{name: "custom port", address: "127.0.0.1:9090", want: "127.0.0.1:9090"},
		{name: "IPv6 loopback", address: "[::1]:9090", want: "[::1]:9090"},
		{name: "IPv4 wildcard", address: "0.0.0.0:9090"},
		{name: "IPv6 wildcard", address: "[::]:9090"},
		{name: "remote address", address: "192.168.1.1:9090"},
		{name: "hostname", address: "localhost:9090"},
		{name: "missing host", address: ":9090"},
		{name: "missing port", address: "127.0.0.1"},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := loadFromLookup(func(key string) (string, bool) {
				if key == "GEMINI_API_KEY" {
					return "synthetic-not-a-real-key", true
				}
				return test.address, key == "AGENT_LISTEN_ADDR"
			})
			if test.want == "" {
				if err == nil {
					t.Fatal("accepted invalid Agent listen address")
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if cfg.ListenAddress != test.want {
				t.Fatalf("got %q, want %q", cfg.ListenAddress, test.want)
			}
		})
	}
}

func TestRuntimeConfig(t *testing.T) {
	for _, test := range []struct {
		name    string
		env     map[string]string
		want    Config
		wantErr string
	}{
		{
			name: "defaults",
			want: Config{Mode: "fixture", ListenAddress: "127.0.0.1:8090"},
		},
		{
			name: "local configuration",
			env: map[string]string{
				"AGENT_MODE":               " local ",
				"AGENT_LISTEN_ADDR":        "127.0.0.1:9090",
				"AGENT_WORKER_URL":         " http://127.0.0.1:8000 ",
				"AGENT_STATE_DATABASE_URL": "postgres://synthetic:password@127.0.0.1/test",
			},
			want: Config{
				Mode: "local", ListenAddress: "127.0.0.1:9090",
				WorkerURL:        "http://127.0.0.1:8000",
				StateDatabaseURL: "postgres://synthetic:password@127.0.0.1/test",
			},
		},
		{
			name: "database connection string preserved",
			env:  map[string]string{"AGENT_STATE_DATABASE_URL": "host=localhost password=' leading and trailing '"},
			want: Config{
				Mode: "fixture", ListenAddress: "127.0.0.1:8090",
				StateDatabaseURL: "host=localhost password=' leading and trailing '",
			},
		},
		{
			name: "empty model key", env: map[string]string{"GEMINI_API_KEY": ""},
			wantErr: "GEMINI_API_KEY",
		},
		{
			name: "blank model key", env: map[string]string{"GEMINI_API_KEY": "   "},
			wantErr: "GEMINI_API_KEY",
		},
		{
			name: "trimmed model key", env: map[string]string{"GEMINI_API_KEY": " synthetic-not-a-real-key "},
			want: Config{Mode: "fixture", ListenAddress: "127.0.0.1:8090"},
		},
		{
			name: "unknown mode", env: map[string]string{"AGENT_MODE": "unknown"},
			wantErr: "AGENT_MODE",
		},
		{
			name: "local requires database", env: map[string]string{"AGENT_MODE": "local"},
			wantErr: "AGENT_STATE_DATABASE_URL",
		},
		{
			name: "local rejects blank database",
			env: map[string]string{
				"AGENT_MODE": "local", "AGENT_STATE_DATABASE_URL": "   ",
			},
			wantErr: "AGENT_STATE_DATABASE_URL",
		},
		{
			name: "local requires worker",
			env: map[string]string{
				"AGENT_MODE": "local", "AGENT_STATE_DATABASE_URL": "synthetic-dsn",
			},
			wantErr: "AGENT_WORKER_URL",
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			cfg, err := loadFromLookup(func(key string) (string, bool) {
				value, ok := test.env[key]
				if key == "GEMINI_API_KEY" && !ok {
					return "synthetic-not-a-real-key", true
				}
				return value, ok
			})
			if test.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), test.wantErr) {
					t.Fatalf("expected %s configuration error, got %v", test.wantErr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			test.want.GeminiAPIKey = "synthetic-not-a-real-key"
			test.want.ModelPolicy = models.ModelPolicy{Default: models.ModelProfile{Name: "gemini-3.8-flash", MaxOutputTokens: 2048}, Router: models.ModelProfile{Name: "gemini-3.5-flash-lite", MaxOutputTokens: 2048}}
			test.want.Limits = models.LLMLimits{MaxModelCalls: 6, MaxToolCalls: 2, MaxStateUpdates: 4, MaxContextBytes: 64 << 10, MaxResponseBytes: 32 << 10, MaxTotalResponseBytes: 256 << 10, MaxResponseEvents: 256}

			if cfg != test.want {
				t.Fatal("loaded configuration does not match configured values")
			}
		})
	}
}

func TestMissingGeminiAPIKey(t *testing.T) {
	_, err := loadFromLookup(func(string) (string, bool) { return "", false })
	if err == nil || !strings.Contains(err.Error(), "GEMINI_API_KEY") {
		t.Fatal("missing Gemini API key did not stop startup")
	}
}

func TestConfiguredModelsAndLimits(t *testing.T) {
	env := map[string]string{
		"GEMINI_API_KEY":      "synthetic-not-a-real-key",
		"AGENT_DEFAULT_MODEL": " operator-default ", "AGENT_ROUTER_MODEL": "operator-router",
		"AGENT_DEFAULT_MAX_OUTPUT_TOKENS": "3000", "AGENT_ROUTER_MAX_OUTPUT_TOKENS": "700",
		"AGENT_MAX_MODEL_CALLS": "8", "AGENT_MAX_TOOL_CALLS": "3", "AGENT_MAX_STATE_UPDATES": "5",
		"AGENT_MAX_CONTEXT_BYTES": "10000", "AGENT_MAX_RESPONSE_BYTES": "1000",
		"AGENT_MAX_TOTAL_RESPONSE_BYTES": "5000", "AGENT_MAX_RESPONSE_EVENTS": "20",
	}
	cfg, err := loadFromLookup(func(key string) (string, bool) { value, ok := env[key]; return value, ok })
	if err != nil {
		t.Fatal(err)
	}
	expectedPolicy := models.ModelPolicy{Default: models.ModelProfile{Name: "operator-default", MaxOutputTokens: 3000}, Router: models.ModelProfile{Name: "operator-router", MaxOutputTokens: 700}}
	expectedLimits := models.LLMLimits{MaxModelCalls: 8, MaxToolCalls: 3, MaxStateUpdates: 5, MaxContextBytes: 10000, MaxResponseBytes: 1000, MaxTotalResponseBytes: 5000, MaxResponseEvents: 20}
	if cfg.ModelPolicy != expectedPolicy || cfg.Limits != expectedLimits {
		t.Fatal("environment settings were not preserved")
	}
}

func TestInvalidModelAndLimitSettings(t *testing.T) {
	keys := []string{"AGENT_DEFAULT_MAX_OUTPUT_TOKENS", "AGENT_ROUTER_MAX_OUTPUT_TOKENS", "AGENT_MAX_MODEL_CALLS", "AGENT_MAX_TOOL_CALLS", "AGENT_MAX_STATE_UPDATES", "AGENT_MAX_CONTEXT_BYTES", "AGENT_MAX_RESPONSE_BYTES", "AGENT_MAX_TOTAL_RESPONSE_BYTES", "AGENT_MAX_RESPONSE_EVENTS"}
	for _, key := range keys {
		for _, invalid := range []string{"", "0", "-1", "1.5", "not-a-number", "2147483648"} {
			t.Run(key+"/"+invalid, func(t *testing.T) {
				_, err := loadFromLookup(func(name string) (string, bool) {
					if name == "GEMINI_API_KEY" {
						return "synthetic-not-a-real-key", true
					}
					return invalid, name == key
				})
				if err == nil || !strings.Contains(err.Error(), key) {
					t.Fatalf("invalid setting was accepted: %v", err)
				}
			})
		}
	}
	for _, key := range []string{"AGENT_DEFAULT_MODEL", "AGENT_ROUTER_MODEL"} {
		_, err := loadFromLookup(func(name string) (string, bool) {
			if name == "GEMINI_API_KEY" {
				return "synthetic-not-a-real-key", true
			}
			return "   ", name == key
		})
		if err == nil || !strings.Contains(err.Error(), key) {
			t.Fatal("blank model name was accepted")
		}
	}
	_, err := loadFromLookup(func(key string) (string, bool) {
		if key == "GEMINI_API_KEY" {
			return "synthetic-not-a-real-key", true
		}
		return "10", key == "AGENT_MAX_TOTAL_RESPONSE_BYTES"
	})
	if err == nil || !strings.Contains(err.Error(), "total response bytes") {
		t.Fatal("inconsistent response limits were accepted")
	}
}
