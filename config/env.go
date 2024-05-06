package config

import (
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	DB     DBConfig
	DBTest TestDBConfig
}

var Envs = initConfig()

func initConfig() Config {
	godotenv.Load()

	var config Config
	config.DB, config.DBTest = initDBConfig()

	return config
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
