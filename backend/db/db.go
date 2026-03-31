package db

import (
	"database/sql"
	"expense-tracker/backend/config"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

type poolConfig struct {
	maxOpenConns    int
	maxIdleConns    int
	connMaxLifetime time.Duration
	connMaxIdleTime time.Duration
}

func defaultPoolConfig() poolConfig {
	return poolConfig{
		maxOpenConns:    25,
		maxIdleConns:    10,
		connMaxLifetime: 30 * time.Minute,
		connMaxIdleTime: 5 * time.Minute,
	}
}

func BuildPostgreSQLDSN(cfg config.Config) string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBPublicHost, cfg.DBPort, cfg.DBName, cfg.DBSSLMode)
}

func configureConnectionPool(db *sql.DB, cfg poolConfig) {
	db.SetMaxOpenConns(cfg.maxOpenConns)
	db.SetMaxIdleConns(cfg.maxIdleConns)
	db.SetConnMaxLifetime(cfg.connMaxLifetime)
	db.SetConnMaxIdleTime(cfg.connMaxIdleTime)
}

func poolConfigFromConfig(cfg config.Config) poolConfig {
	pool := defaultPoolConfig()

	if cfg.DBMaxOpenConns > 0 {
		pool.maxOpenConns = cfg.DBMaxOpenConns
	}
	if cfg.DBMaxIdleConns > 0 {
		pool.maxIdleConns = cfg.DBMaxIdleConns
	}
	if cfg.DBConnMaxLifetimeSeconds > 0 {
		pool.connMaxLifetime = time.Duration(cfg.DBConnMaxLifetimeSeconds) * time.Second
	}
	if cfg.DBConnMaxIdleTimeSeconds > 0 {
		pool.connMaxIdleTime = time.Duration(cfg.DBConnMaxIdleTimeSeconds) * time.Second
	}

	return pool
}

func NewPostgreSQLStorage(cfg config.Config) (*sql.DB, error) {
	db, err := sql.Open("pgx", BuildPostgreSQLDSN(cfg))
	if err != nil {
		return nil, err
	}

	configureConnectionPool(db, poolConfigFromConfig(cfg))
	return db, nil
}
