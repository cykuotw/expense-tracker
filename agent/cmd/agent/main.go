package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"expense-tracker/agent/internal/httpapi"
	"expense-tracker/agent/internal/models"
	agentruntime "expense-tracker/agent/internal/runtime"
	"expense-tracker/agent/internal/tools"
)

const listenAddress = "127.0.0.1:8090"

func main() {
	if err := run(); err != nil {
		slog.Error("agent startup failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	if err := validateStartup(os.Args[1:]); err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	agent, err := agentruntime.New(ctx, agentruntime.Config{
		Models:  models.FixtureFactory,
		Balance: tools.FixtureBalanceClient{},
	})
	if err != nil {
		return err
	}
	for _, diagnostic := range agent.Diagnostics() {
		slog.Info("agent configuration",
			"skill", diagnostic.Skill,
			"model", diagnostic.Model.Resolved,
			"fallback", diagnostic.Model.Fallback,
			"skillVersion", diagnostic.SkillVersion,
		)
	}

	server := &http.Server{
		Addr:              listenAddress,
		Handler:           httpapi.NewLocalFixture(agent),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      70 * time.Second,
		IdleTimeout:       30 * time.Second,
		MaxHeaderBytes:    8 << 10,
	}

	go func() {
		<-ctx.Done()
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	}()
	slog.Info("synthetic agent fixture listening", "address", listenAddress)
	err = server.ListenAndServe()
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}

func validateStartup(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("agent does not accept command-line arguments")
	}
	if _, ok := os.LookupEnv("AWS_LAMBDA_RUNTIME_API"); ok {
		return fmt.Errorf("local fixtures cannot run in Lambda")
	}
	return nil
}
