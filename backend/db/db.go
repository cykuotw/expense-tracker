package db

import (
	"database/sql"
	"expense-tracker/backend/config"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgreSQLStorage(cfg config.Config) (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.DBUser, cfg.DBPassword, cfg.DBPublicHost, cfg.DBPort, cfg.DBName)

	db, err := sql.Open("pgx", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	return db, nil
}
