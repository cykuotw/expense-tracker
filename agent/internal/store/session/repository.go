// Package session provides injected session and execution storage services.
package session

import (
	"context"
	"time"

	"expense-tracker/agent/internal/models"
)

type Repository interface {
	ResolveThread(context.Context, string, string, string, bool) (models.Key, error)
	ResolveRun(context.Context, models.Key, string, string) (string, error)
	ListConversations(context.Context, string, string) ([]models.Conversation, error)
	BeginUI(context.Context, models.Key, models.UIRef, string, any) (models.Run, error)
	Begin(context.Context, models.Key, string, any) (models.Run, error)
	Append(context.Context, models.Run, models.Entry) (string, time.Time, error)
	Finish(context.Context, models.Run, string, string) error
	PersistAndEmitOutput(context.Context, models.Run, models.Output, func(models.Output) error) error
	ListSessions(context.Context, string, string) ([]models.Conversation, error)
	ListRuns(context.Context, models.Key) ([]models.Run, error)
	ReadRunOutput(context.Context, models.Key, string, string) ([]models.Output, error)
	SoftDelete(context.Context, models.Key) error
}
