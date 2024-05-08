package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DBPublicHost string
	DBPort       string
	DBUser       string
	DBPassword   string
	DBName       string
}

var Envs = initConfig()

func initConfig() Config {
	godotenv.Load()

	return Config{
		DBPublicHost: getEnv("DB_PUBLIC_HOST", "localhost"),
		DBPort:       getEnv("DB_PORT", "5432"),
		DBUser:       getEnv("DB_USER", "tracker"),
		DBPassword:   getEnv("DB_PASSWORD", "mypassword"),
		DBName:       getEnv("DB_NAME", "mydb"),
	}
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
