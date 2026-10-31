// Package handlers translates HTTP requests and service results without depending on the Agent runtime.
package handlers

import (
	"context"

	"expense-tracker/agent/internal/models"
)

type Application interface {
	Run(context.Context, models.ChatRequest, func(models.Output) error) error
	ListConversations(context.Context) ([]models.Conversation, error)
	ListRuns(context.Context, string) ([]models.Run, error)
	ReadRunOutput(context.Context, string, string, string) ([]models.Output, error)
	DeleteConversation(context.Context, string) error
}

type Handler struct {
	application Application
}

func New(application Application) *Handler {
	return &Handler{application: application}
}
