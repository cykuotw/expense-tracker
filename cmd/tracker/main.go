package main

import (
	"expense-tracker/cmd/tracker/api"
	"expense-tracker/config"
	"expense-tracker/db"
	"log"
)

func main() {
	cfg := config.Envs

	db, err := db.NewPostgreSQLStorage(cfg)
	if err != nil {
		log.Fatal(err)
	}
	err = db.Ping()
	if err != nil {
		log.Fatal(err)
	}

	server := api.NewAPIServer("localhost:8080", nil)
	if err := server.Run(); err != nil {
		log.Fatal(err)
	}
}
