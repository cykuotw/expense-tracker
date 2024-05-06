package db

import (
	"database/sql"
	config "expense-tracker/config"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
)

func NewPostgreSQLStorage(cfg config.DBConfig) (*sql.DB, error) {
	psqlInfo := fmt.Sprintf("postgres://%s:%s@%s:%s/%s",
		cfg.User, cfg.Password, cfg.PublicHost, cfg.Port, cfg.DBName)

	db, err := sql.Open("pgx", psqlInfo)
	if err != nil {
		log.Fatal(err)
	}

	return db, nil
}
