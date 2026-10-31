package chat_test

import (
	"context"
	"iter"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"expense-tracker/agent/internal/fixtures"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/runtime"
	sessionstore "expense-tracker/agent/internal/store/session"
	"expense-tracker/agent/internal/tools"

	"github.com/stretchr/testify/require"
	"google.golang.org/adk/v2/model"
	"google.golang.org/adk/v2/session"
	"google.golang.org/genai"
)

type blockedThreadRepository struct {
	sessionstore.Repository
	entered chan struct{}
	release chan struct{}
	calls   atomic.Int32
}

func (repo *blockedThreadRepository) ResolveThread(ctx context.Context, _, _, _ string, _ bool) (models.Key, error) {
	repo.calls.Add(1)
	select {
	case repo.entered <- struct{}{}:
	default:
	}
	select {
	case <-repo.release:
		return models.Key{}, sessionstore.ErrStorage
	case <-ctx.Done():
		return models.Key{}, ctx.Err()
	}
}

func TestPersistentChatUsesProcessLocalAdmission(t *testing.T) {
	repo := &blockedThreadRepository{entered: make(chan struct{}, 1), release: make(chan struct{})}
	app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(), Models: fixtures.ModelFactory, Tools: testCatalog(t, &authenticatedBalance{actor: "owner", expiry: time.Now().Add(time.Hour)}), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(), Authenticate: (&authenticatedBalance{actor: "owner", expiry: time.Now().Add(time.Hour)}).Authenticate, Sessions: session.InMemoryService(), Records: repo})
	require.NoError(t, err)
	request := models.ChatRequest{ConversationID: "thread", RunID: "ui-run", MessageID: "one", Text: "hello"}
	done := make(chan error, 1)
	go func() { done <- app.Run(t.Context(), request, func(models.Output) error { return nil }) }()
	<-repo.entered
	other := request
	other.MessageID, other.RunID = "two", "second-run"
	err = app.Run(t.Context(), other, func(models.Output) error { return nil })
	require.Equal(t, models.Limit, models.PublicCode(err))
	require.Equal(t, int32(1), repo.calls.Load(), "busy threads are rejected before storage access")
	close(repo.release)
	require.ErrorIs(t, <-done, sessionstore.ErrStorage)
	err = app.Run(t.Context(), request, func(models.Output) error { return nil })
	require.ErrorIs(t, err, sessionstore.ErrStorage, "a pre-start storage failure releases resend admission")
	require.Equal(t, int32(2), repo.calls.Load())
}

func TestConcurrentRunLimitsAndRelease(t *testing.T) {
	entered := make(chan struct{}, 4)
	release := make(chan struct{})
	factory := func(context.Context, string, string) (model.LLM, error) {
		return modelFunc(func(
			ctx context.Context,
			_ *model.LLMRequest,
			_ bool,
		) iter.Seq2[*model.LLMResponse, error] {
			return func(yield func(*model.LLMResponse, error) bool) {
				entered <- struct{}{}
				select {
				case <-release:
				case <-ctx.Done():
					yield(nil, ctx.Err())
					return
				}
				yield(&model.LLMResponse{
					Content:      genai.NewContentFromText(`{"kind":"unsupported"}`, "model"),
					TurnComplete: true,
				}, nil)
			}
		}), nil
	}
	app, err := runtime.New(t.Context(), runtime.Dependencies{ModelPolicy: fixtures.ModelPolicy(), Limits: fixtures.Limits(),
		Models: factory,
		Tools:  testCatalog(t, fixtures.BalanceClient{}), CompletionChecks: tools.CompletionChecks(),
		RouterTools:          tools.RouterTools(),
		InstructionProviders: tools.InstructionProviders(),
		AfterModelCallbacks:  tools.AfterModelCallbacks(),
	})
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	results := make(chan error, 4)
	for _, id := range []string{"one", "two", "three", "four"} {
		wg.Go(func() {
			results <- app.Run(t.Context(), models.ChatRequest{
				ConversationID: id,
				MessageID:      "one",
				Text:           "unsupported",
			}, func(models.Output) error { return nil })
		})
	}
	for range 4 {
		<-entered
	}
	for _, id := range []string{"one", "five"} {
		err := app.Run(t.Context(), models.ChatRequest{
			ConversationID: id,
			MessageID:      "two",
			Text:           "unsupported",
		}, func(models.Output) error { return nil })
		if models.PublicCode(err) != models.Limit {
			t.Error("concurrent run was accepted", err)
		}
	}
	close(release)
	wg.Wait()
	close(results)
	for err := range results {
		if err != nil {
			t.Fatal(err)
		}
	}
	if err = app.Run(t.Context(), models.ChatRequest{
		ConversationID: "one",
		MessageID:      "two",
		Text:           "unsupported",
	}, func(models.Output) error { return nil }); err != nil {
		t.Fatal("run slot not released", err)
	}
}
