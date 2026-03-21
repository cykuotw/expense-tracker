package main

import (
	"context"
	"expense-tracker/backend/cmd/tracker/api"
	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"
)

func closeDBPool(storage interface{ Close() error }) {
	if err := storage.Close(); err != nil {
		log.Printf("failed to close db pool: %v", err)
	}
}

func main() {
	cfg := config.Envs

	storage, err := dbstore.NewPostgreSQLStorage(cfg)
	if err != nil {
		log.Fatal(err)
	}

	if err := storage.Ping(); err != nil {
		closeDBPool(storage)
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	apiServer := api.NewAPIServer(config.Envs.BackendURL, storage)
	go func() {
		if err := apiServer.Run(); err != nil && err != http.ErrServerClosed {
			log.Fatal(err)
		}
	}()

	<-ctx.Done()

	stop()
	log.Println("Shutting down system")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := apiServer.Shutdown(shutdownCtx); err != nil {
		closeDBPool(storage)
		log.Fatal(err)
	}
	closeDBPool(storage)
	log.Println("	API Server is shut down")
}
