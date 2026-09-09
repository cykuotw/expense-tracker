package config

import (
	"net/http"
	"strings"
	"testing"
)

func validReleaseConfig(profile RequestServerProfile) Config {
	provided := make(map[string]bool)
	for _, key := range []string{
		"BACKEND_URL", "FRONTEND_ORIGIN", "DB_PUBLIC_HOST", "DB_PORT", "DB_USER",
		"DB_PASSWORD", "DB_NAME", "DB_SSLMODE", "DB_MAX_OPEN_CONNS",
		"DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME_SECONDS",
		"DB_CONN_MAX_IDLE_TIME_SECONDS", "JWT_SECRET", "JWT_EXP",
		"REFRESH_JWT_SECRET", "REFRESH_JWT_EXP", "EXPENSES_PER_PAGE",
		"CORS_ALLOWED_ORIGINS", "CORS_ALLOW_CREDENTIALS", "AUTH_COOKIE_SECURE",
		"AUTH_COOKIE_SAME_SITE", "GOOGLE_OAUTH_ENABLED", "GOOGLE_EXCHANGE_MODE",
	} {
		provided[key] = true
	}
	mode := GoogleExchangeInProcess
	if profile == RequestServerServerless {
		mode = GoogleExchangeUpstreamVerified
	}
	return Config{
		Mode:                          "release",
		BackendURL:                    "127.0.0.1:8000",
		FrontendOrigin:                "https://app.example.com",
		DBPublicHost:                  "10.0.0.2",
		DBPort:                        "5432",
		DBUser:                        "expense_runtime",
		DBPassword:                    "database-password-not-a-placeholder",
		DBName:                        "expense_tracker",
		DBSSLMode:                     "disable",
		DBMaxOpenConns:                2,
		DBMaxIdleConns:                1,
		DBConnMaxLifetimeSeconds:      300,
		DBConnMaxIdleTimeSeconds:      60,
		JWTSecret:                     strings.Repeat("a", 32),
		JWTExpirationInSeconds:        900,
		RefreshJWTSecret:              strings.Repeat("b", 32),
		RefreshJWTExpirationInSeconds: 604_800,
		GoogleOAuthEnabled:            true,
		GoogleClientId:                "client.apps.googleusercontent.com",
		GoogleExchangeMode:            mode,
		ExpensesPerPage:               25,
		CORSAllowedOrigins:            []string{"https://app.example.com"},
		CORSAllowCredentials:          true,
		AuthCookieSecure:              true,
		AuthCookieSameSite:            http.SameSiteLaxMode,
		provided:                      provided,
	}
}

func TestValidateRequestServerAcceptsSupportedProfiles(t *testing.T) {
	loaded, err := loadFromLookup(func(string) (string, bool) { return "", false })
	if err != nil {
		t.Fatalf("valid local defaults were rejected: %v", err)
	}
	if loaded.Mode != "debug" || loaded.FrontendOrigin != "http://localhost:5173" || loaded.AuthCookieSecure {
		t.Fatal("unexpected local debug mode, origin, or cookie defaults")
	}
	debug := Config{Mode: "debug", GoogleExchangeMode: GoogleExchangeInProcess}
	if err := ValidateRequestServer(debug, RequestServerStandalone); err != nil {
		t.Fatalf("valid debug configuration rejected: %v", err)
	}
	for _, profile := range []RequestServerProfile{RequestServerStandalone, RequestServerServerless} {
		if err := ValidateRequestServer(validReleaseConfig(profile), profile); err != nil {
			t.Fatalf("valid %s configuration rejected: %v", profile, err)
		}
	}
	standalone := validReleaseConfig(RequestServerStandalone)
	standalone.GoogleOAuthEnabled = false
	standalone.GoogleClientId = ""
	if err := ValidateRequestServer(standalone, RequestServerStandalone); err != nil {
		t.Fatalf("release configuration with Google OAuth disabled was rejected: %v", err)
	}
	noneCookie := validReleaseConfig(RequestServerStandalone)
	noneCookie.AuthCookieSameSite = http.SameSiteNoneMode
	if err := ValidateRequestServer(noneCookie, RequestServerStandalone); err != nil {
		t.Fatalf("secure SameSite=None configuration was rejected: %v", err)
	}
	boundaries := validReleaseConfig(RequestServerStandalone)
	boundaries.JWTExpirationInSeconds = MinAccessTokenLifetimeSeconds
	boundaries.RefreshJWTExpirationInSeconds = MaxRefreshTokenLifetimeSeconds
	boundaries.DBMaxOpenConns = MaxDBConnections
	boundaries.DBMaxIdleConns = MaxDBConnections
	boundaries.DBConnMaxLifetimeSeconds = 0
	boundaries.DBConnMaxIdleTimeSeconds = MaxDBConnectionDurationSeconds
	boundaries.ExpensesPerPage = MaxExpensesPerPage
	if err := ValidateRequestServer(boundaries, RequestServerStandalone); err != nil {
		t.Fatalf("valid release boundary values were rejected: %v", err)
	}
}

func TestValidateRequestServerRequiresEveryExplicitReleaseValue(t *testing.T) {
	cfg := validReleaseConfig(RequestServerStandalone)
	for key := range cfg.provided {
		t.Run(key, func(t *testing.T) {
			candidate := validReleaseConfig(RequestServerStandalone)
			candidate.provided[key] = false
			err := ValidateRequestServer(candidate, RequestServerStandalone)
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("expected missing %s error, got %v", key, err)
			}
		})
	}
}

func TestValidateRequestServerRejectsInvalidReleaseConfiguration(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		mutate func(*Config)
	}{
		{"missing database host", "DB_PUBLIC_HOST", func(cfg *Config) { cfg.provided["DB_PUBLIC_HOST"] = false }},
		{"development database host", "DB_PUBLIC_HOST", func(cfg *Config) { cfg.DBPublicHost = "localhost" }},
		{"development database user", "DB_USER", func(cfg *Config) { cfg.DBUser = "tracker" }},
		{"development database name", "DB_NAME", func(cfg *Config) { cfg.DBName = "mydb" }},
		{"invalid database port", "DB_PORT", func(cfg *Config) { cfg.DBPort = "70000" }},
		{"unsupported ssl mode", "DB_SSLMODE", func(cfg *Config) { cfg.DBSSLMode = "bogus" }},
		{"placeholder database password", "DB_PASSWORD", func(cfg *Config) { cfg.DBPassword = "CHANGE_ME" }},
		{"short access secret", "JWT_SECRET", func(cfg *Config) { cfg.JWTSecret = "short" }},
		{"placeholder access secret", "JWT_SECRET", func(cfg *Config) { cfg.JWTSecret = "secretstring" }},
		{"short refresh secret", "REFRESH_JWT_SECRET", func(cfg *Config) { cfg.RefreshJWTSecret = "short" }},
		{"identical secrets", "JWT_SECRET", func(cfg *Config) { cfg.RefreshJWTSecret = cfg.JWTSecret }},
		{"short access lifetime", "JWT_EXP", func(cfg *Config) { cfg.JWTExpirationInSeconds = MinAccessTokenLifetimeSeconds - 1 }},
		{"long access lifetime", "JWT_EXP", func(cfg *Config) { cfg.JWTExpirationInSeconds = MaxAccessTokenLifetimeSeconds + 1 }},
		{"short refresh lifetime", "REFRESH_JWT_EXP", func(cfg *Config) { cfg.RefreshJWTExpirationInSeconds = MinRefreshTokenLifetimeSeconds - 1 }},
		{"long refresh lifetime", "REFRESH_JWT_EXP", func(cfg *Config) { cfg.RefreshJWTExpirationInSeconds = MaxRefreshTokenLifetimeSeconds + 1 }},
		{"unordered lifetimes", "REFRESH_JWT_EXP", func(cfg *Config) { cfg.RefreshJWTExpirationInSeconds = cfg.JWTExpirationInSeconds }},
		{"zero open connections", "DB_MAX_OPEN_CONNS", func(cfg *Config) { cfg.DBMaxOpenConns = 0 }},
		{"too many open connections", "DB_MAX_OPEN_CONNS", func(cfg *Config) { cfg.DBMaxOpenConns = MaxDBConnections + 1 }},
		{"too many idle connections", "DB_MAX_IDLE_CONNS", func(cfg *Config) { cfg.DBMaxIdleConns = 3 }},
		{"negative connection lifetime", "DB_CONN_MAX_LIFETIME_SECONDS", func(cfg *Config) { cfg.DBConnMaxLifetimeSeconds = -1 }},
		{"long idle duration", "DB_CONN_MAX_IDLE_TIME_SECONDS", func(cfg *Config) { cfg.DBConnMaxIdleTimeSeconds = MaxDBConnectionDurationSeconds + 1 }},
		{"zero pagination", "EXPENSES_PER_PAGE", func(cfg *Config) { cfg.ExpensesPerPage = 0 }},
		{"excessive pagination", "EXPENSES_PER_PAGE", func(cfg *Config) { cfg.ExpensesPerPage = MaxExpensesPerPage + 1 }},
		{"http frontend", "FRONTEND_ORIGIN", func(cfg *Config) { cfg.FrontendOrigin = "http://app.example.com" }},
		{"frontend path", "FRONTEND_ORIGIN", func(cfg *Config) { cfg.FrontendOrigin = "https://app.example.com/path" }},
		{"frontend query", "FRONTEND_ORIGIN", func(cfg *Config) { cfg.FrontendOrigin = "https://app.example.com?query=yes" }},
		{"frontend fragment", "FRONTEND_ORIGIN", func(cfg *Config) { cfg.FrontendOrigin = "https://app.example.com#fragment" }},
		{"frontend user info", "FRONTEND_ORIGIN", func(cfg *Config) { cfg.FrontendOrigin = "https://user@app.example.com" }},
		{"frontend invalid port", "FRONTEND_ORIGIN", func(cfg *Config) { cfg.FrontendOrigin = "https://app.example.com:70000" }},
		{"frontend wildcard host", "FRONTEND_ORIGIN", func(cfg *Config) { cfg.FrontendOrigin = "https://*.example.com" }},
		{"wildcard cors", "CORS_ALLOWED_ORIGINS", func(cfg *Config) { cfg.CORSAllowedOrigins = []string{"*"} }},
		{"cors mismatch", "CORS_ALLOWED_ORIGINS", func(cfg *Config) { cfg.CORSAllowedOrigins = []string{"https://other.example.com"} }},
		{"credentials disabled", "CORS_ALLOW_CREDENTIALS", func(cfg *Config) { cfg.CORSAllowCredentials = false }},
		{"insecure cookie", "AUTH_COOKIE_SECURE", func(cfg *Config) { cfg.AuthCookieSecure = false }},
		{"invalid listener", "BACKEND_URL", func(cfg *Config) { cfg.BackendURL = "localhost" }},
		{"invalid listener port", "BACKEND_URL", func(cfg *Config) { cfg.BackendURL = "localhost:70000" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := validReleaseConfig(RequestServerStandalone)
			test.mutate(&cfg)
			err := ValidateRequestServer(cfg, RequestServerStandalone)
			if err == nil || !strings.Contains(err.Error(), test.key) {
				t.Fatalf("expected error containing %s, got %v", test.key, err)
			}
			if strings.Contains(err.Error(), cfg.DBPassword) || strings.Contains(err.Error(), cfg.JWTSecret) ||
				strings.Contains(err.Error(), cfg.RefreshJWTSecret) {
				t.Fatalf("validation error exposed secret material: %v", err)
			}
		})
	}
}

func TestEffectiveModeUsesAuthoritativeBuildMode(t *testing.T) {
	original := BuildMode
	BuildMode = "release"
	t.Cleanup(func() { BuildMode = original })
	if mode := effectiveMode("debug"); mode != "release" {
		t.Fatalf("expected release build mode, got %q", mode)
	}
}

func TestLoadStrictRejectsMalformedExplicitValues(t *testing.T) {
	tests := map[string]string{
		"JWT_EXP":                "one hour",
		"AUTH_COOKIE_SECURE":     "yes",
		"AUTH_COOKIE_SAME_SITE":  "sometimes",
		"CORS_ALLOW_CREDENTIALS": "1",
		"DB_MAX_OPEN_CONNS":      "many",
		"GOOGLE_OAUTH_ENABLED":   "TRUE",
		"GOOGLE_EXCHANGE_MODE":   "",
		"MODE":                   "production",
	}
	for key, value := range tests {
		t.Run(key, func(t *testing.T) {
			lookup := func(name string) (string, bool) {
				if name == key {
					return value, true
				}
				return "", false
			}
			_, err := loadFromLookup(lookup)
			if err == nil || !strings.Contains(err.Error(), key) {
				t.Fatalf("expected key-specific parse error, got %v", err)
			}
		})
	}
}

func TestLoadedServerlessWorkerEnvironmentPassesRuntimeValidation(t *testing.T) {
	values := map[string]string{
		"MODE": "release", "API_URL": "/api/v0",
		"FRONTEND_ORIGIN":        "https://app.example.com",
		"CORS_ALLOWED_ORIGINS":   "https://app.example.com",
		"CORS_ALLOW_CREDENTIALS": "true", "AUTH_COOKIE_SECURE": "true",
		"AUTH_COOKIE_SAME_SITE": "lax", "DB_PUBLIC_HOST": "10.0.0.2",
		"DB_PORT": "5432", "DB_USER": "expense_runtime",
		"DB_PASSWORD": "database-password-not-a-placeholder",
		"DB_NAME":     "expense_tracker", "DB_SSLMODE": "disable",
		"DB_MAX_OPEN_CONNS": "2", "DB_MAX_IDLE_CONNS": "1",
		"DB_CONN_MAX_LIFETIME_SECONDS":  "300",
		"DB_CONN_MAX_IDLE_TIME_SECONDS": "60",
		"JWT_SECRET":                    strings.Repeat("a", 32), "JWT_EXP": "900",
		"REFRESH_JWT_SECRET": strings.Repeat("b", 32), "REFRESH_JWT_EXP": "604800",
		"EXPENSES_PER_PAGE": "25", "GOOGLE_OAUTH_ENABLED": "true",
		"GOOGLE_CLIENT_ID":     "client.apps.googleusercontent.com",
		"GOOGLE_EXCHANGE_MODE": "upstream_verified",
	}
	lookup := func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
	cfg, err := loadFromLookup(lookup)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateRequestServer(cfg, RequestServerServerless); err != nil {
		t.Fatalf("generated Worker environment was rejected: %v", err)
	}
}
