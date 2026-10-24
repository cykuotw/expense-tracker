// Package runtime assembles the transport-independent, read-only agent runtime.
package runtime

import (
	"cmp"
	"context"
	"io/fs"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode/utf8"

	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/tools"
	"github.com/google/uuid"
	"google.golang.org/adk/v2/runner"
	"google.golang.org/adk/v2/session"
)

const (
	MaxTextBytes = 8192

	maxConversations        = 64
	maxConversationMessages = 64
	maxActiveRuns           = 4
	sessionAppName          = "expense-agent"
	runTimeout              = 60 * time.Second
)

var identifier = regexp.MustCompile(`^[a-zA-Z0-9_-]{1,128}$`)

func validID(id string) bool {
	return identifier.MatchString(id)
}

type Request struct {
	ConversationID string `json:"conversationId"`
	MessageID      string `json:"messageId"`
	Text           string `json:"text"`
}

func (request Request) Validate() error {
	if !validID(request.ConversationID) || !validID(request.MessageID) {
		return &tools.Error{Code: tools.Validation}
	}

	invalidText := !utf8.ValidString(request.Text) ||
		strings.TrimSpace(request.Text) == "" ||
		len(request.Text) > MaxTextBytes
	if invalidText {
		return &tools.Error{Code: tools.Validation}
	}

	return nil
}

// Output is an internal projection. It carries correlation and version evidence,
// never raw ADK content, function arguments, provider errors, or thought parts.
type Output struct {
	Kind                 string
	RunID                string
	ConversationID       string
	ClientMessageID      string
	InvocationID         string
	Skill                string
	SkillVersion         string
	Model                string
	ProviderModelVersion string
	Delta                string
}

type Config struct {
	Models       models.Factory
	Balance      tools.BalanceClient
	Sessions     session.Service
	Skills       fs.FS
	DefaultModel string
	Logger       *slog.Logger
}

type Diagnostic struct {
	Skill        string
	Model        models.Resolution
	SkillVersion string
}

type conversation struct {
	busy     bool
	messages map[string]bool
}

type App struct {
	runner      *runner.Runner
	diagnostics map[string]Diagnostic
	agentSkills map[string]string
	logger      *slog.Logger

	mu            sync.Mutex
	conversations map[string]*conversation
	active        int
}

func (app *App) Diagnostics() []Diagnostic {
	result := make([]Diagnostic, 0, len(app.diagnostics))
	for _, diagnostic := range app.diagnostics {
		result = append(result, diagnostic)
	}
	slices.SortFunc(result, func(left, right Diagnostic) int { return cmp.Compare(left.Skill, right.Skill) })
	return result
}

// Run serializes one conversation and rejects duplicates. In-memory retention
// has explicit local limits; persistence and authenticated ownership are separate.
func (app *App) Run(ctx context.Context, request Request, emit func(Output) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := request.Validate(); err != nil {
		return err
	}
	if err := app.reserveRun(request); err != nil {
		return err
	}
	defer app.releaseRun(request.ConversationID)

	ctx, cancel := context.WithTimeout(tools.WithReadTracking(models.WithBudget(ctx)), runTimeout)
	defer cancel()

	output := Output{
		Kind:            "started",
		RunID:           uuid.New().String(),
		ConversationID:  request.ConversationID,
		ClientMessageID: request.MessageID,
	}
	ctx = withRunLogger(ctx, app.logger, request, output.RunID)
	if err := emit(output); err != nil {
		return err
	}

	if err := app.execute(ctx, request, output, emit); err != nil {
		return err
	}

	output.Kind = "finished"
	return emit(output)
}

// reserveRun keeps admission checks and state changes under the same lock.
func (app *App) reserveRun(request Request) error {
	app.mu.Lock()
	defer app.mu.Unlock()

	current := app.conversations[request.ConversationID]
	if current == nil {
		if len(app.conversations) >= maxConversations {
			return &tools.Error{Code: tools.Limit}
		}
		current = &conversation{messages: map[string]bool{}}
		app.conversations[request.ConversationID] = current
	}

	atLimit := app.active >= maxActiveRuns ||
		len(current.messages) >= maxConversationMessages
	if current.busy || atLimit || current.messages[request.MessageID] {
		return &tools.Error{Code: tools.Limit}
	}

	current.busy = true
	current.messages[request.MessageID] = true
	app.active++
	return nil
}

func (app *App) releaseRun(conversationID string) {
	app.mu.Lock()
	defer app.mu.Unlock()

	app.conversations[conversationID].busy = false
	app.active--
}
