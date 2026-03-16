package config

import (
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	Mode string

	BackendURL     string
	FrontendOrigin string
	APIPath        string

	DBPublicHost string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string

	JWTSecret                     string
	JWTExpirationInSeconds        int64
	RefreshJWTSecret              string
	RefreshJWTExpirationInSeconds int64

	GoogleClientId     string
	GoogleClientSecret string
	GoogleCallbackUrl  string

	ThirdPartySessionSecret string
	ThirdPartySessionMaxAge int64

	ExpensesPerPage int64

	CORSAllowedOrigins   []string
	CORSAllowCredentials bool

	AuthCookieDomain string
	AuthCookieSecure bool
}

var Envs = initConfig()

// passed during build time
var BuildMode string

func initConfig() Config {
	loadLocalEnv()

	mode := getEnv("MODE", "debug")
	if BuildMode != "" {
		mode = BuildMode
	}

	return Config{
		Mode: mode,

		BackendURL:     getEnv("BACKEND_URL", "127.0.0.1:8000"),
		FrontendOrigin: normalizeOrigin(getEnv("FRONTEND_ORIGIN", "http://localhost:5173")),
		APIPath:        getEnv("API_URL", ""),

		DBPublicHost: getEnv("DB_PUBLIC_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "tracker"),
		DBPassword:   getEnv("DB_PASSWORD", "mypassword"),
		DBName:       getEnv("DB_NAME", "mydb"),

		JWTSecret:                     getEnv("JWT_SECRET", "secretstring"),
		JWTExpirationInSeconds:        getEnvInt("JWT_EXP", 3600*24*7),
		RefreshJWTSecret:              getEnv("REFRESH_JWT_SECRET", getEnv("JWT_SECRET", "secretstring")),
		RefreshJWTExpirationInSeconds: getEnvInt("REFRESH_JWT_EXP", 3600*24*30),

		GoogleClientId:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleCallbackUrl:  getEnv("GOOGLE_CALLBACK_URL", ""),

		ThirdPartySessionSecret: getEnv("THIRD_PARTY_SESSION_SECRET", ""),
		ThirdPartySessionMaxAge: getEnvInt("THIRD_PARTY_SESSION_MAX_AGE", 3600*24*7),

		ExpensesPerPage: getEnvInt("EXPENSES_PER_PAGE", 25),

		CORSAllowedOrigins: parseOrigins(
			getEnv("CORS_ALLOWED_ORIGINS", "http://localhost:5173"),
		),
		CORSAllowCredentials: getEnvBool("CORS_ALLOW_CREDENTIALS", false),

		AuthCookieDomain: getEnv("AUTH_COOKIE_DOMAIN", ""),
		AuthCookieSecure: getEnvBool("AUTH_COOKIE_SECURE", false),
	}
}

func loadLocalEnv() {
	candidates := []string{
		".env",
		"backend/.env",
	}

	for _, path := range candidates {
		if err := godotenv.Load(path); err == nil {
			return
		}
	}
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func getEnvInt(key string, fallback int64) int64 {
	if value, ok := os.LookupEnv(key); ok {
		num, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return fallback
		}
		return num
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	if value, ok := os.LookupEnv(key); ok {
		switch value {
		case "True", "true", "1", "yes":
			return true
		case "False", "false", "0", "no":
			return false
		}
	}
	return fallback
}

func parseOrigins(value string) []string {
	if value == "" {
		return nil
	}

	parts := strings.Split(value, ",")
	origins := make([]string, 0, len(parts))
	for _, part := range parts {
		origin := normalizeOrigin(strings.TrimSpace(part))
		if origin == "" {
			continue
		}
		origins = append(origins, origin)
	}

	return origins
}

func normalizeOrigin(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	if strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") {
		return value
	}
	return "http://" + value
}
