package main

import (
	"context"
	"expense-tracker/cmd/tracker/api"
	"expense-tracker/config"
	"expense-tracker/db"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
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

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	apiServer := api.NewAPIServer(config.Envs.BackendURL, db)
	go func() {
		if err := apiServer.Run(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()

	stop()
	log.Println("Shutting down system")

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
	log.Println("\tAPI Server is shut down")
}
