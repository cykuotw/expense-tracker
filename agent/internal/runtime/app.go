// Package runtime composes the ADK engine and chat/history application services.
package runtime

import (
	"context"
	"fmt"
	"io/fs"
	"log/slog"

	"expense-tracker/agent/internal/llm"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/runtime/engine"
	"expense-tracker/agent/internal/runtime/services/chat"
	"expense-tracker/agent/internal/runtime/services/conversation"
	sessionstore "expense-tracker/agent/internal/store/session"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/agent/llmagent"
	"google.golang.org/adk/v2/session"
	"google.golang.org/adk/v2/tool"
)

// Dependencies supplies the services and inputs used to assemble the application.
type Dependencies struct {
	Models               llm.Factory
	Tools                map[string]tool.Tool
	CompletionChecks     map[string]func(agent.Context) error
	RouterTools          []string
	InstructionProviders map[string]llmagent.InstructionProvider
	AfterModelCallbacks  map[string]llmagent.AfterModelCallback
	Authenticate         func(context.Context) (models.Identity, error)
	Sessions             session.Service
	Skills               fs.FS
	ModelPolicy          models.ModelPolicy
	Limits               models.LLMLimits
	Logger               *slog.Logger
	Records              sessionstore.Repository
}

type SkillMetadata = models.SkillMetadata

// App exposes chat and history through the existing transport-independent API.
type App struct {
	engine        *engine.Engine
	chat          *chat.Service
	conversations *conversation.Service
}

// New assembles ADK, chat and history services after checking durable wiring.
func New(ctx context.Context, deps Dependencies) (*App, error) {
	// Reject incomplete durable wiring before the engine can default to memory.
	if deps.Records != nil && deps.Sessions == nil {
		return nil, fmt.Errorf("durable execution requires explicit persistent sessions")
	}
	if deps.Records != nil && deps.Authenticate == nil {
		return nil, fmt.Errorf("durable execution requires explicit authentication")
	}
	core, err := engine.New(ctx, engine.Dependencies{
		Models: deps.Models, Tools: deps.Tools, CompletionChecks: deps.CompletionChecks,
		RouterTools: deps.RouterTools, InstructionProviders: deps.InstructionProviders,
		AfterModelCallbacks: deps.AfterModelCallbacks, Sessions: deps.Sessions,
		Skills: deps.Skills, ModelPolicy: deps.ModelPolicy, Limits: deps.Limits, Logger: deps.Logger,
	})
	if err != nil {
		return nil, err
	}
	chats, err := chat.New(chat.Dependencies{Engine: core, Records: deps.Records, Authenticate: deps.Authenticate, Limits: deps.Limits})
	if err != nil {
		return nil, err
	}
	history := conversation.New(engine.AppName, deps.Records, chats.AuthenticatedOwner, chats.IsRunning)
	return &App{engine: core, chat: chats, conversations: history}, nil
}

func (app *App) Run(ctx context.Context, request models.ChatRequest, emit func(models.Output) error) error {
	return app.chat.Run(ctx, request, emit)
}

func (app *App) SkillMetadata() []SkillMetadata { return app.engine.SkillMetadata() }

func (app *App) AuthenticatedOwner(ctx context.Context) (string, error) {
	return app.chat.AuthenticatedOwner(ctx)
}

func (app *App) IsRunning(owner, conversation, run string) bool {
	return app.chat.IsRunning(owner, conversation, run)
}

func (app *App) ListConversations(ctx context.Context) ([]models.Conversation, error) {
	return app.conversations.ListConversations(ctx)
}

func (app *App) ListRuns(ctx context.Context, conversation string) ([]models.Run, error) {
	return app.conversations.ListRuns(ctx, conversation)
}

func (app *App) ReadRunOutput(ctx context.Context, conversation, run, after string) ([]models.Output, error) {
	return app.conversations.ReadRunOutput(ctx, conversation, run, after)
}

func (app *App) DeleteConversation(ctx context.Context, conversation string) error {
	return app.conversations.DeleteConversation(ctx, conversation)
}
