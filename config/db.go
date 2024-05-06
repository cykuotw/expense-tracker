package config

type DBConfig struct {
	PublicHost string
	Port       string
	User       string
	Password   string
	DBName     string
}

type TestDBConfig struct {
	PublicHost string
	Port       string
	User       string
	Password   string
	DBName     string
}

func initDBConfig() (DBConfig, TestDBConfig) {
	return DBConfig{
			PublicHost: getEnv("DB_PUBLIC_HOST", "localhost"),
			Port:       getEnv("DB_PORT", "5432"),
			User:       getEnv("DB_USER", "tracker"),
			Password:   getEnv("DB_PASSWORD", "mypassword"),
			DBName:     getEnv("DB_NAME", "mydb"),
		}, TestDBConfig{
			PublicHost: getEnv("DB_TEST_HOST", "localhost"),
			Port:       getEnv("DB_TEST_PORT", "5432"),
			User:       getEnv("DB_TEST_USER", "trackertest"),
			Password:   getEnv("DB_TEST_PASSWORD", "mytestpassword"),
			DBName:     getEnv("DB_TEST_NAME", "mytestdb"),
		}
}
