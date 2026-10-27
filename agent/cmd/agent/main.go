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

	"expense-tracker/agent/internal/api"
	"expense-tracker/agent/internal/clients/worker"
	"expense-tracker/agent/internal/models"
	agentruntime "expense-tracker/agent/internal/runtime"
	"expense-tracker/agent/internal/tools"
	"expense-tracker/internal/localonly"
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

	settings := agentruntime.Config{
		Models:  models.FixtureFactory,
		Balance: tools.FixtureBalanceClient{},
	}
	local := os.Getenv("AGENT_MODE") == "local"
	if local {
		client, err := worker.NewLocal(os.Getenv("AGENT_WORKER_URL"))
		if err != nil {
			return fmt.Errorf("invalid local Worker configuration")
		}
		settings.Balance = client
	}
	agent, err := agentruntime.New(ctx, settings)
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

	handler := api.NewLocalFixture(agent)
	if local {
		handler, err = api.NewLocal(agent)
		if err != nil {
			return err
		}
	}
	server := &http.Server{
		Addr:              listenAddress,
		Handler:           handler,
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
	slog.Info("local agent listening", "address", listenAddress, "workerEnabled", local)
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
	if err := localonly.CheckRuntime(); err != nil {
		return err
	}
	switch os.Getenv("AGENT_MODE") {
	case "", "fixture":
	case "local":
		client, err := worker.NewLocal(os.Getenv("AGENT_WORKER_URL"))
		if err != nil || client == nil {
			return fmt.Errorf("local mode requires a loopback AGENT_WORKER_URL")
		}
	default:
		return fmt.Errorf("unsupported AGENT_MODE")
	}
	return nil
}
