package config

import (
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"
)

type RequestServerProfile string

const (
	RequestServerStandalone RequestServerProfile = "standalone"
	RequestServerServerless RequestServerProfile = "serverless"

	MinAccessTokenLifetimeSeconds  int64 = 60
	MaxAccessTokenLifetimeSeconds  int64 = 86_400
	MinRefreshTokenLifetimeSeconds int64 = 300
	MaxRefreshTokenLifetimeSeconds int64 = 31_536_000
	MaxDBConnections                     = 100
	MaxDBConnectionDurationSeconds int64 = 86_400
	MaxExpensesPerPage             int64 = 1_000
)

var strictIntegerKeys = []struct {
	name   string
	target func(*Config, int64)
}{
	{"DB_MAX_OPEN_CONNS", func(cfg *Config, value int64) { cfg.DBMaxOpenConns = int(value) }},
	{"DB_MAX_IDLE_CONNS", func(cfg *Config, value int64) { cfg.DBMaxIdleConns = int(value) }},
	{"DB_CONN_MAX_LIFETIME_SECONDS", func(cfg *Config, value int64) { cfg.DBConnMaxLifetimeSeconds = value }},
	{"DB_CONN_MAX_IDLE_TIME_SECONDS", func(cfg *Config, value int64) { cfg.DBConnMaxIdleTimeSeconds = value }},
	{"JWT_EXP", func(cfg *Config, value int64) { cfg.JWTExpirationInSeconds = value }},
	{"REFRESH_JWT_EXP", func(cfg *Config, value int64) { cfg.RefreshJWTExpirationInSeconds = value }},
	{"EXPENSES_PER_PAGE", func(cfg *Config, value int64) { cfg.ExpensesPerPage = value }},
}

var strictBooleanKeys = []struct {
	name   string
	target func(*Config, bool)
}{
	{"GOOGLE_OAUTH_ENABLED", func(cfg *Config, value bool) { cfg.GoogleOAuthEnabled = value }},
	{"CORS_ALLOW_CREDENTIALS", func(cfg *Config, value bool) { cfg.CORSAllowCredentials = value }},
	{"AUTH_COOKIE_SECURE", func(cfg *Config, value bool) { cfg.AuthCookieSecure = value }},
}

func Load() (Config, error) {
	loadLocalEnv()
	return loadFromLookup(os.LookupEnv)
}

func loadFromLookup(lookup func(string) (string, bool)) (Config, error) {
	cfg := Config{
		Mode:                          "debug",
		BackendURL:                    "127.0.0.1:8000",
		FrontendOrigin:                "http://localhost:5173",
		DBPublicHost:                  "localhost",
		DBPort:                        "5432",
		DBUser:                        "tracker",
		DBPassword:                    "mypassword",
		DBName:                        "mydb",
		DBSSLMode:                     "disable",
		DBMaxOpenConns:                25,
		DBMaxIdleConns:                10,
		DBConnMaxLifetimeSeconds:      1_800,
		DBConnMaxIdleTimeSeconds:      300,
		JWTSecret:                     "secretstring",
		JWTExpirationInSeconds:        604_800,
		RefreshJWTSecret:              "secretstring",
		RefreshJWTExpirationInSeconds: 2_592_000,
		GoogleExchangeMode:            GoogleExchangeInProcess,
		ExpensesPerPage:               25,
		CORSAllowedOrigins:            []string{"http://localhost:5173"},
		AuthCookieSameSite:            http.SameSiteLaxMode,
	}

	setString := func(key string, target *string) {
		if value, ok := lookup(key); ok {
			*target = strings.TrimSpace(value)
		}
	}
	setString("MODE", &cfg.Mode)
	setString("BACKEND_URL", &cfg.BackendURL)
	setString("API_URL", &cfg.APIPath)
	setString("DB_PUBLIC_HOST", &cfg.DBPublicHost)
	setString("DB_PORT", &cfg.DBPort)
	setString("DB_USER", &cfg.DBUser)
	setString("DB_PASSWORD", &cfg.DBPassword)
	setString("DB_NAME", &cfg.DBName)
	setString("DB_SSLMODE", &cfg.DBSSLMode)
	setString("JWT_SECRET", &cfg.JWTSecret)
	if value, ok := lookup("REFRESH_JWT_SECRET"); ok {
		cfg.RefreshJWTSecret = strings.TrimSpace(value)
	} else {
		cfg.RefreshJWTSecret = cfg.JWTSecret
	}
	setString("GOOGLE_CLIENT_ID", &cfg.GoogleClientId)
	setString("AUTH_COOKIE_DOMAIN", &cfg.AuthCookieDomain)
	setString("WEB_PUSH_VAPID_PUBLIC_KEY", &cfg.WebPushVAPIDPublicKey)
	if value, ok := lookup("FRONTEND_ORIGIN"); ok {
		cfg.FrontendOrigin = normalizeOrigin(value)
	}
	if value, ok := lookup("CORS_ALLOWED_ORIGINS"); ok {
		cfg.CORSAllowedOrigins = parseOrigins(value)
	}
	if value, ok := lookup("GOOGLE_EXCHANGE_MODE"); ok {
		if strings.TrimSpace(value) == "" {
			cfg.GoogleExchangeMode = ""
		} else {
			cfg.GoogleExchangeMode = normalizeGoogleExchangeMode(value)
		}
	}
	cfg.Mode = effectiveMode(cfg.Mode)

	cfg.provided = make(map[string]bool)

	for _, key := range []string{
		"MODE", "BACKEND_URL", "FRONTEND_ORIGIN", "API_URL", "DB_PUBLIC_HOST",
		"DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"JWT_SECRET", "REFRESH_JWT_SECRET", "GOOGLE_CLIENT_ID",
		"GOOGLE_EXCHANGE_MODE", "CORS_ALLOWED_ORIGINS", "AUTH_COOKIE_DOMAIN",
		"AUTH_COOKIE_SAME_SITE",
	} {
		_, cfg.provided[key] = lookup(key)
	}

	for _, item := range strictIntegerKeys {
		raw, ok := lookup(item.name)
		cfg.provided[item.name] = ok
		if !ok {
			continue
		}
		value, err := strconv.ParseInt(strings.TrimSpace(raw), 10, 64)
		if err != nil {
			return Config{}, fmt.Errorf("%s must be an integer", item.name)
		}
		item.target(&cfg, value)
	}

	for _, item := range strictBooleanKeys {
		raw, ok := lookup(item.name)
		cfg.provided[item.name] = ok
		if !ok {
			continue
		}
		normalized := strings.TrimSpace(raw)
		value, err := strconv.ParseBool(normalized)
		if err != nil || (normalized != "true" && normalized != "false") {
			return Config{}, fmt.Errorf("%s must be true or false", item.name)
		}
		item.target(&cfg, value)
	}

	if raw, ok := lookup("AUTH_COOKIE_SAME_SITE"); ok {
		switch strings.ToLower(strings.TrimSpace(raw)) {
		case "default":
			cfg.AuthCookieSameSite = http.SameSiteDefaultMode
		case "lax":
			cfg.AuthCookieSameSite = http.SameSiteLaxMode
		case "strict":
			cfg.AuthCookieSameSite = http.SameSiteStrictMode
		case "none":
			cfg.AuthCookieSameSite = http.SameSiteNoneMode
		default:
			return Config{}, fmt.Errorf("AUTH_COOKIE_SAME_SITE must be default, lax, strict, or none")
		}
	}

	if err := validateCommon(cfg); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func Apply(cfg Config) {
	Envs = cfg
}

func ValidateRequestServer(cfg Config, profile RequestServerProfile) error {
	if err := validateCommon(cfg); err != nil {
		return err
	}
	if profile != RequestServerStandalone && profile != RequestServerServerless {
		return fmt.Errorf("unknown request-server validation profile %q", profile)
	}
	if profile == RequestServerServerless && cfg.Mode != "release" {
		return fmt.Errorf("MODE must be release for the serverless request server")
	}
	if cfg.Mode != "release" {
		return nil
	}

	required := []string{
		"DB_PUBLIC_HOST", "DB_PORT", "DB_USER", "DB_PASSWORD", "DB_NAME", "DB_SSLMODE",
		"DB_MAX_OPEN_CONNS", "DB_MAX_IDLE_CONNS", "DB_CONN_MAX_LIFETIME_SECONDS",
		"DB_CONN_MAX_IDLE_TIME_SECONDS", "JWT_SECRET", "JWT_EXP", "REFRESH_JWT_SECRET",
		"REFRESH_JWT_EXP", "EXPENSES_PER_PAGE", "FRONTEND_ORIGIN", "CORS_ALLOWED_ORIGINS",
		"CORS_ALLOW_CREDENTIALS", "AUTH_COOKIE_SECURE", "AUTH_COOKIE_SAME_SITE",
		"GOOGLE_OAUTH_ENABLED", "GOOGLE_EXCHANGE_MODE",
	}
	if profile == RequestServerStandalone {
		required = append(required, "BACKEND_URL")
	}
	for _, key := range required {
		if !cfg.provided[key] {
			return fmt.Errorf("%s must be explicitly configured in release mode", key)
		}
	}

	for _, item := range []struct {
		key   string
		value string
	}{
		{"DB_PUBLIC_HOST", cfg.DBPublicHost},
		{"DB_USER", cfg.DBUser},
		{"DB_PASSWORD", cfg.DBPassword},
		{"DB_NAME", cfg.DBName},
	} {
		if strings.TrimSpace(item.value) == "" {
			return fmt.Errorf("%s must not be empty", item.key)
		}
	}
	if isPlaceholder(cfg.DBPublicHost, "localhost", "127.0.0.1") {
		return fmt.Errorf("DB_PUBLIC_HOST must not use a development default")
	}
	if isPlaceholder(cfg.DBUser, "tracker") {
		return fmt.Errorf("DB_USER must not use a development default")
	}
	if isPlaceholder(cfg.DBPassword, "mypassword") || hasPlaceholderPrefix(cfg.DBPassword) {
		return fmt.Errorf("DB_PASSWORD must not use a placeholder")
	}
	if isPlaceholder(cfg.DBName, "mydb") {
		return fmt.Errorf("DB_NAME must not use a development default")
	}
	if err := validatePort("DB_PORT", cfg.DBPort); err != nil {
		return err
	}
	if !slices.Contains([]string{"disable", "allow", "prefer", "require", "verify-ca", "verify-full"}, cfg.DBSSLMode) {
		return fmt.Errorf("DB_SSLMODE must be a supported PostgreSQL SSL mode")
	}
	if profile == RequestServerStandalone {
		if err := validateListener(cfg.BackendURL); err != nil {
			return err
		}
	}
	if err := validateSecret("JWT_SECRET", cfg.JWTSecret); err != nil {
		return err
	}
	if err := validateSecret("REFRESH_JWT_SECRET", cfg.RefreshJWTSecret); err != nil {
		return err
	}
	if cfg.JWTSecret == cfg.RefreshJWTSecret {
		return fmt.Errorf("JWT_SECRET and REFRESH_JWT_SECRET must be different")
	}
	if err := validateRange("JWT_EXP", cfg.JWTExpirationInSeconds, MinAccessTokenLifetimeSeconds, MaxAccessTokenLifetimeSeconds); err != nil {
		return err
	}
	if err := validateRange("REFRESH_JWT_EXP", cfg.RefreshJWTExpirationInSeconds, MinRefreshTokenLifetimeSeconds, MaxRefreshTokenLifetimeSeconds); err != nil {
		return err
	}
	if cfg.RefreshJWTExpirationInSeconds <= cfg.JWTExpirationInSeconds {
		return fmt.Errorf("REFRESH_JWT_EXP must be greater than JWT_EXP")
	}
	if cfg.DBMaxOpenConns < 1 || cfg.DBMaxOpenConns > MaxDBConnections {
		return fmt.Errorf("DB_MAX_OPEN_CONNS must be between 1 and %d", MaxDBConnections)
	}
	if cfg.DBMaxIdleConns < 0 || cfg.DBMaxIdleConns > cfg.DBMaxOpenConns {
		return fmt.Errorf("DB_MAX_IDLE_CONNS must be between 0 and DB_MAX_OPEN_CONNS")
	}
	if err := validateRange("DB_CONN_MAX_LIFETIME_SECONDS", cfg.DBConnMaxLifetimeSeconds, 0, MaxDBConnectionDurationSeconds); err != nil {
		return err
	}
	if err := validateRange("DB_CONN_MAX_IDLE_TIME_SECONDS", cfg.DBConnMaxIdleTimeSeconds, 0, MaxDBConnectionDurationSeconds); err != nil {
		return err
	}
	if err := validateRange("EXPENSES_PER_PAGE", cfg.ExpensesPerPage, 1, MaxExpensesPerPage); err != nil {
		return err
	}
	if err := validateHTTPSOrigin("FRONTEND_ORIGIN", cfg.FrontendOrigin); err != nil {
		return err
	}
	if !cfg.CORSAllowCredentials {
		return fmt.Errorf("CORS_ALLOW_CREDENTIALS must be true in release mode")
	}
	if len(cfg.CORSAllowedOrigins) == 0 {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must not be empty in release mode")
	}
	for _, origin := range cfg.CORSAllowedOrigins {
		if origin == "*" {
			return fmt.Errorf("CORS_ALLOWED_ORIGINS must not contain a wildcard")
		}
		if err := validateHTTPSOrigin("CORS_ALLOWED_ORIGINS", origin); err != nil {
			return err
		}
	}
	if !slices.Contains(cfg.CORSAllowedOrigins, cfg.FrontendOrigin) {
		return fmt.Errorf("CORS_ALLOWED_ORIGINS must contain FRONTEND_ORIGIN")
	}
	if !cfg.AuthCookieSecure {
		return fmt.Errorf("AUTH_COOKIE_SECURE must be true in release mode")
	}
	if cfg.AuthCookieSameSite == http.SameSiteNoneMode && !cfg.AuthCookieSecure {
		return fmt.Errorf("AUTH_COOKIE_SAME_SITE=None requires AUTH_COOKIE_SECURE=true")
	}
	if profile == RequestServerServerless && cfg.GoogleExchangeMode != GoogleExchangeUpstreamVerified {
		return fmt.Errorf("GOOGLE_EXCHANGE_MODE must be upstream_verified for the serverless request server")
	}
	return nil
}

func validateCommon(cfg Config) error {
	if cfg.Mode != "debug" && cfg.Mode != "test" && cfg.Mode != "release" {
		return fmt.Errorf("MODE must be debug, test, or release")
	}
	return validateGoogleOAuthConfig(cfg)
}

func validateSecret(key, value string) error {
	if len([]byte(value)) < 32 || hasPlaceholderPrefix(value) || isPlaceholder(value, "secretstring") {
		return fmt.Errorf("%s must be a non-placeholder secret of at least 32 bytes", key)
	}
	return nil
}

func hasPlaceholderPrefix(value string) bool {
	upper := strings.ToUpper(strings.TrimSpace(value))
	return strings.HasPrefix(upper, "REPLACE") || strings.HasPrefix(upper, "CHANGE") || strings.HasPrefix(upper, "EXAMPLE")
}

func isPlaceholder(value string, placeholders ...string) bool {
	for _, placeholder := range placeholders {
		if strings.EqualFold(strings.TrimSpace(value), placeholder) {
			return true
		}
	}
	return false
}

func validatePort(key, value string) error {
	port, err := strconv.Atoi(value)
	if err != nil || port < 1 || port > 65_535 {
		return fmt.Errorf("%s must be an integer between 1 and 65535", key)
	}
	return nil
}

func validateListener(value string) error {
	_, port, err := net.SplitHostPort(value)
	if err != nil {
		return fmt.Errorf("BACKEND_URL must contain a host and port")
	}
	return validatePort("BACKEND_URL port", port)
}

func validateRange(key string, value, minimum, maximum int64) error {
	if value < minimum || value > maximum {
		return fmt.Errorf("%s must be between %d and %d", key, minimum, maximum)
	}
	return nil
}

func validateHTTPSOrigin(key, value string) error {
	parsed, err := url.Parse(value)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || parsed.User != nil ||
		parsed.Path != "" || parsed.RawQuery != "" || parsed.Fragment != "" || parsed.Opaque != "" {
		return fmt.Errorf("%s must be an absolute HTTPS origin without user info, path, query, or fragment", key)
	}
	if strings.Contains(parsed.Hostname(), "*") {
		return fmt.Errorf("%s must not contain a wildcard", key)
	}
	if port := parsed.Port(); port != "" {
		if err := validatePort(key+" port", port); err != nil {
			return err
		}
	}
	return nil
}
