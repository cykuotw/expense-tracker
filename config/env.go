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

	TestDBPublicHost string
	TestDBPort       string
	TestDBUser       string
	TestDBPassword   string
	TestDBName       string
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

		TestDBPublicHost: getEnv("DB_TEST_HOST", "localhost"),
		TestDBPort:       getEnv("DB_TEST_PORT", "5432"),
		TestDBUser:       getEnv("DB_TEST_USER", "trackertest"),
		TestDBPassword:   getEnv("DB_TEST_PASSWORD", "mytestpassword"),
		TestDBName:       getEnv("DB_TEST_NAME", "mytestdb"),
	}
}

func getEnv(key string, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}
