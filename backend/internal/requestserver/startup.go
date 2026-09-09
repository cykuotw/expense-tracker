package requestserver

import (
	"expense-tracker/backend/config"
	"fmt"
)

type Database interface {
	Ping() error
	Close() error
}

func OpenDatabase[T Database](cfg config.Config, profile config.RequestServerProfile, open func(config.Config) (T, error)) (T, error) {
	var zero T
	if err := config.ValidateRequestServer(cfg, profile); err != nil {
		return zero, fmt.Errorf("invalid request-server configuration: %w", err)
	}

	config.Apply(cfg)
	storage, err := open(cfg)
	if err != nil {
		return zero, err
	}
	if err := storage.Ping(); err != nil {
		_ = storage.Close()
		return zero, err
	}
	return storage, nil
}
