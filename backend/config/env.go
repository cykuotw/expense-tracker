package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Mode string

	BackendURL       string
	FrontendURL      string
	FrontendReactURL string
	APIPath          string

	DBPublicHost string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string

	JWTSecret              string
	JWTExpirationInSeconds int64
	RefreshJWTSecret       string
	RefreshJWTExpirationInSeconds int64

	GoogleClientId     string
	GoogleClientSecret string
	GoogleCallbackUrl  string

	ThirdPartySessionSecret string
	ThirdPartySessionMaxAge int64

	ExpensesPerPage int64

	CORSFrontendOrigin   string
	CORSAllowCredentials bool
}

var Envs = initConfig()

// passed during build time
var BuildMode string

func initConfig() Config {
	godotenv.Load()

	mode := getEnv("MODE", "debug")
	if BuildMode != "" {
		mode = BuildMode
	}

	return Config{
		Mode: mode,

		BackendURL:       getEnv("BACKEND_URL", "localhost:8000"),
		FrontendURL:      getEnv("FRONTEND_URL", "localhost:8050"),
		FrontendReactURL: getEnv("FRONTEND_REACT_URL", "localhost:5173"),
		APIPath:          getEnv("API_URL", ""),

		DBPublicHost: getEnv("DB_PUBLIC_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "tracker"),
		DBPassword:   getEnv("DB_PASSWORD", "mypassword"),
		DBName:       getEnv("DB_NAME", "mydb"),

		JWTSecret:              getEnv("JWT_SECRET", "secretstring"),
		JWTExpirationInSeconds: getEnvInt("JWT_EXP", 3600*24*7),
		RefreshJWTSecret:       getEnv("REFRESH_JWT_SECRET", getEnv("JWT_SECRET", "secretstring")),
		RefreshJWTExpirationInSeconds: getEnvInt("REFRESH_JWT_EXP", 3600*24*30),

		GoogleClientId:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleCallbackUrl:  getEnv("GOOGLE_CALLBACK_URL", ""),

		ThirdPartySessionSecret: getEnv("THIRD_PARTY_SESSION_SECRET", ""),
		ThirdPartySessionMaxAge: getEnvInt("THIRD_PARTY_SESSION_MAX_AGE", 3600*24*7),

		ExpensesPerPage: getEnvInt("EXPENSES_PER_PAGE", 25),

		CORSFrontendOrigin:   getEnv("CORS_FRONTEND_ORIGIN", "localhost:8050"),
		CORSAllowCredentials: getEnvBool("CORS_ALLOW_CREDENTIALS", false),
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
