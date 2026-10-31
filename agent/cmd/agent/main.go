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

	"expense-tracker/agent/config"
	"expense-tracker/agent/internal/adapters/expenseapi"
	"expense-tracker/agent/internal/api"
	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/llm"
	agentruntime "expense-tracker/agent/internal/runtime"
	sessionstore "expense-tracker/agent/internal/store/session"
	"expense-tracker/agent/internal/tools"
	"expense-tracker/agent/internal/tools/functiontool/getbalance"

	"google.golang.org/genai"
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		slog.Error("agent startup failed", "error", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	if len(args) != 0 {
		return fmt.Errorf("agent does not accept command-line arguments")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	deps := agentruntime.Dependencies{
		ModelPolicy: cfg.ModelPolicy, Limits: cfg.Limits,
		Models:               llm.NewGeminiFactory(genai.ClientConfig{APIKey: cfg.GeminiAPIKey, Backend: genai.BackendGeminiAPI}),
		CompletionChecks:     tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(),
	}
	var balance getbalance.Client = fixtures.BalanceClient{}
	isLocal := cfg.Mode == "local"
	if isLocal {
		client, err := expenseapi.NewLocal(cfg.WorkerURL)
		if err != nil {
			return fmt.Errorf("local mode requires a loopback AGENT_WORKER_URL")
		}
		balance = client
		deps.Authenticate = client.Authenticate
		store, err := sessionstore.Open(ctx, cfg.StateDatabaseURL)
		if err != nil {
			return fmt.Errorf("isolated Agent state database unavailable")
		}
		defer store.Close()
		deps.Sessions = store
		deps.Records = store
	}
	deps.Tools, err = tools.NewCatalog(balance)
	if err != nil {
		return err
	}
	handler, err := buildHandler(ctx, deps, isLocal)
	if err != nil {
		return err
	}
	return serve(ctx, handler, cfg)
}

func buildHandler(ctx context.Context, deps agentruntime.Dependencies, isLocal bool) (http.Handler, error) {
	agent, err := agentruntime.New(ctx, deps)
	if err != nil {
		return nil, err
	}
	for _, metadata := range agent.SkillMetadata() {
		slog.Info("agent configuration",
			"skill", metadata.Name,
			"model", metadata.ModelSelection.Selected,
			"fallback", metadata.ModelSelection.Fallback,
			"skillVersion", metadata.Version,
		)
	}

	if isLocal {
		return api.NewLocalHandler(agent)
	}
	return api.NewLocalFixtureHandler(agent), nil
}

func serve(ctx context.Context, handler http.Handler, cfg config.Config) error {
	server := api.NewHTTPServer(cfg.ListenAddress, handler)

	shutdownDone := make(chan struct{})
	stopShutdown := context.AfterFunc(ctx, func() {
		defer close(shutdownDone)
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = server.Shutdown(shutdown)
	})
	slog.Info("local agent listening", "address", cfg.ListenAddress, "workerEnabled", cfg.Mode == "local")
	err := server.ListenAndServe()
	if !stopShutdown() {
		<-shutdownDone
	}
	if errors.Is(err, http.ErrServerClosed) {
		return nil
	}
	return err
}
