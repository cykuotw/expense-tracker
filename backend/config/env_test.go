package config

import "testing"

func TestGetEnvBool(t *testing.T) {
	t.Run("accepts lowercase true and false", func(t *testing.T) {
		t.Setenv("TEST_BOOL", "true")
		if value := getEnvBool("TEST_BOOL", false); !value {
			t.Fatal("expected lowercase true to parse as true")
		}

		t.Setenv("TEST_BOOL", "false")
		if value := getEnvBool("TEST_BOOL", true); value {
			t.Fatal("expected lowercase false to parse as false")
		}
	})

	t.Run("accepts trimmed lowercase values", func(t *testing.T) {
		t.Setenv("TEST_BOOL", " true ")
		if value := getEnvBool("TEST_BOOL", false); !value {
			t.Fatal("expected trimmed lowercase true to parse as true")
		}
	})

	t.Run("falls back for legacy non-contract values", func(t *testing.T) {
		t.Setenv("TEST_BOOL", "True")
		if value := getEnvBool("TEST_BOOL", false); value {
			t.Fatal("expected legacy capitalized true to fall back")
		}

		t.Setenv("TEST_BOOL", "1")
		if value := getEnvBool("TEST_BOOL", false); value {
			t.Fatal("expected numeric true to fall back")
		}

		t.Setenv("TEST_BOOL", "yes")
		if value := getEnvBool("TEST_BOOL", true); !value {
			t.Fatal("expected invalid legacy value to return fallback")
		}
	})
}

func TestNormalizeGoogleExchangeMode(t *testing.T) {
	t.Run("defaults blank to inprocess", func(t *testing.T) {
		if mode := normalizeGoogleExchangeMode("   "); mode != GoogleExchangeInProcess {
			t.Fatalf("expected inprocess mode, got %q", mode)
		}
	})

	t.Run("normalizes case and whitespace", func(t *testing.T) {
		if mode := normalizeGoogleExchangeMode(" InProcess "); mode != GoogleExchangeInProcess {
			t.Fatalf("expected inprocess mode, got %q", mode)
		}
	})
}

func TestValidateGoogleOAuthConfig(t *testing.T) {
	t.Run("accepts configured inprocess mode", func(t *testing.T) {
		validateGoogleOAuthConfig(Config{
			GoogleOAuthEnabled: true,
			GoogleClientId:     "client-id",
			GoogleExchangeMode: GoogleExchangeInProcess,
		})
	})

	t.Run("rejects invalid mode", func(t *testing.T) {
		defer func() {
			if recover() == nil {
				t.Fatal("expected panic for invalid google exchange mode")
			}
		}()

		validateGoogleOAuthConfig(Config{
			GoogleOAuthEnabled: true,
			GoogleClientId:     "client-id",
			GoogleExchangeMode: GoogleExchangeMode("bogus"),
		})
	})

	t.Run("allows missing google oauth config when exchange mode is unset or defaulted", func(t *testing.T) {
		validateGoogleOAuthConfig(Config{
			GoogleOAuthEnabled: false,
			GoogleClientId:     "",
			GoogleExchangeMode: GoogleExchangeInProcess,
		})
	})
}

func TestGoogleExchangeModeIs(t *testing.T) {
	cfg := Config{
		GoogleOAuthEnabled: true,
		GoogleClientId:     "client-id",
		GoogleExchangeMode: GoogleExchangeUpstreamVerified,
	}

	if !cfg.GoogleExchangeModeIs(GoogleExchangeUpstreamVerified) {
		t.Fatal("expected upstream_verified mode to match")
	}

	if cfg.GoogleExchangeModeIs(GoogleExchangeInProcess) {
		t.Fatal("did not expect inprocess mode to match")
	}
}
