package db

import (
	"testing"
	"time"

	"expense-tracker/backend/config"

	"github.com/stretchr/testify/assert"
)

func TestDefaultPoolConfig(t *testing.T) {
	cfg := defaultPoolConfig()

	assert.Equal(t, 25, cfg.maxOpenConns)
	assert.Equal(t, 10, cfg.maxIdleConns)
	assert.Equal(t, 30*time.Minute, cfg.connMaxLifetime)
	assert.Equal(t, 5*time.Minute, cfg.connMaxIdleTime)
}

func TestPoolConfigFromConfigUsesOverrides(t *testing.T) {
	cfg := poolConfigFromConfig(config.Config{
		DBMaxOpenConns:           5,
		DBMaxIdleConns:           3,
		DBConnMaxLifetimeSeconds: 45,
		DBConnMaxIdleTimeSeconds: 15,
	})

	assert.Equal(t, 5, cfg.maxOpenConns)
	assert.Equal(t, 3, cfg.maxIdleConns)
	assert.Equal(t, 45*time.Second, cfg.connMaxLifetime)
	assert.Equal(t, 15*time.Second, cfg.connMaxIdleTime)
}

func TestNewPostgreSQLStorageConfiguresConnectionPool(t *testing.T) {
	storage, err := NewPostgreSQLStorage(config.Config{
		DBUser:       "tracker",
		DBPassword:   "password",
		DBPublicHost: "localhost",
		DBPort:       "5432",
		DBName:       "tracker",
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		_ = storage.Close()
	})

	assert.Equal(t, defaultPoolConfig().maxOpenConns, storage.Stats().MaxOpenConnections)
}

func TestPoolConfigFromConfigFallsBackToDefaults(t *testing.T) {
	cfg := poolConfigFromConfig(config.Config{})

	assert.Equal(t, defaultPoolConfig(), cfg)
}

func TestBuildPostgreSQLDSN(t *testing.T) {
	dsn := BuildPostgreSQLDSN(config.Config{
		DBUser:       "tracker",
		DBPassword:   "password",
		DBPublicHost: "db.example.com",
		DBPort:       "5432",
		DBName:       "expenses",
		DBSSLMode:    "require",
	})

	assert.Equal(t, "postgres://tracker:password@db.example.com:5432/expenses?sslmode=require", dsn)
}
