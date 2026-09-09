package main

import (
	"context"
	"expense-tracker/backend/config"
	dbstore "expense-tracker/backend/db"
	"expense-tracker/backend/internal/requestserver"
	trackerapp "expense-tracker/backend/internal/tracker"
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
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	storage, err := requestserver.OpenDatabase(cfg, config.RequestServerStandalone, dbstore.NewPostgreSQLStorage)
	if err != nil {
		log.Fatal(err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	handler := trackerapp.NewHandler(storage)
	apiServer := trackerapp.NewHTTPServer(cfg.BackendURL, handler)
	go func() {
		log.Println("API Server Listening on", cfg.BackendURL)
		if err := apiServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
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
