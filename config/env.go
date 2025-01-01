package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	Mode string

	BackendURL  string
	FrontendURL string
	APIPath     string

	DBPublicHost string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string

	JWTSecret              string
	JWTExpirationInSeconds int64

	GoogleClientId     string
	GoogleClientSecret string
	GoogleCallbackUrl  string

	ThirdPartySessionSecret string
	ThirdPartySessionMaxAge int64

	ExpensesPerPage int64
}

var Envs = initConfig()

func initConfig() Config {
	godotenv.Load()

	return Config{
		Mode: getEnv("MODE", "debug"),

		BackendURL:  getEnv("BACKEND_URL", "localhost:8000"),
		FrontendURL: getEnv("FRONTEND_URL", "localhost:8050"),
		APIPath:     getEnv("API_URL", ""),

		DBPublicHost: getEnv("DB_PUBLIC_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "tracker"),
		DBPassword:   getEnv("DB_PASSWORD", "mypassword"),
		DBName:       getEnv("DB_NAME", "mydb"),

		JWTSecret:              getEnv("JWT_SECRET", "secretstring"),
		JWTExpirationInSeconds: getEnvInt("JWT_EXP", 3600*24*7),

		GoogleClientId:     getEnv("GOOGLE_CLIENT_ID", ""),
		GoogleClientSecret: getEnv("GOOGLE_CLIENT_SECRET", ""),
		GoogleCallbackUrl:  getEnv("GOOGLE_CALLBACK_URL", ""),

		ThirdPartySessionSecret: getEnv("THIRD_PARTY_SESSION_SECRET", ""),
		ThirdPartySessionMaxAge: getEnvInt("THIRD_PARTY_SESSION_MAX_AGE", 3600*24*7),

		ExpensesPerPage: getEnvInt("EXPENSES_PER_PAGE", 25),
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
