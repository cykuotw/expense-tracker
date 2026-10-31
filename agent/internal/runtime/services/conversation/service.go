// Package conversation reads and deletes owned recorded conversations.
package conversation

import (
	"context"
	"time"
	"uuid"

	"expense-tracker/agent/internal/models"
	sessionstore "expense-tracker/agent/internal/store/session"
)

// Service manages persisted conversation history without running agents.
type Service struct {
	appName      string
	records      sessionstore.Repository
	resolveOwner func(context.Context) (string, error)
	isRunning    func(owner, conversation, run string) bool
}

// New injects storage, authenticated ownership and process-local run status.
func New(appName string, records sessionstore.Repository, resolveOwner func(context.Context) (string, error), isRunning func(string, string, string) bool) *Service {
	return &Service{appName: appName, records: records, resolveOwner: resolveOwner, isRunning: isRunning}
}

func (service *Service) owner(ctx context.Context) (string, error) {
	if service.records == nil || service.resolveOwner == nil {
		return "", &models.Error{Code: models.Unavailable}
	}
	return service.resolveOwner(ctx)
}

func (service *Service) sessionKey(ctx context.Context, owner, conversation string) (models.Key, error) {
	return service.records.ResolveThread(ctx, service.appName, owner, conversation, false)
}

func (service *Service) ListConversations(ctx context.Context) ([]models.Conversation, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	owner, err := service.owner(ctx)
	if err != nil {
		return nil, err
	}
	return service.records.ListConversations(ctx, service.appName, owner)
}

func (service *Service) ListRuns(ctx context.Context, conversation string) ([]models.Run, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if !models.ValidID(conversation) {
		return nil, &models.Error{Code: models.Validation}
	}
	owner, err := service.owner(ctx)
	if err != nil {
		return nil, err
	}
	key, err := service.sessionKey(ctx, owner, conversation)
	if err != nil {
		return nil, err
	}
	runs, err := service.records.ListRuns(ctx, key)
	if err != nil {
		return nil, err
	}
	for i := range runs {
		if runs[i].UI.ThreadID != conversation || runs[i].UI.RunID == "" {
			return nil, sessionstore.ErrStorage
		}
		runs[i].ID = runs[i].UI.RunID
		if runs[i].Status == "running" && (service.isRunning == nil || !service.isRunning(owner, conversation, runs[i].UI.RunID)) {
			runs[i].Status = "unknown"
		}
	}
	return runs, nil
}

func (service *Service) ReadRunOutput(ctx context.Context, conversation, run, after string) ([]models.Output, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if !models.ValidID(conversation) || !models.ValidID(run) {
		return nil, &models.Error{Code: models.Validation}
	}
	if after != "" {
		if _, err := uuid.Parse(after); err != nil {
			return nil, &models.Error{Code: models.Validation}
		}
	}
	owner, err := service.owner(ctx)
	if err != nil {
		return nil, err
	}
	key, err := service.sessionKey(ctx, owner, conversation)
	if err != nil {
		return nil, err
	}
	internalRun, err := service.records.ResolveRun(ctx, key, conversation, run)
	if err != nil {
		return nil, err
	}
	return service.records.ReadRunOutput(ctx, key, internalRun, after)
}

func (service *Service) DeleteConversation(ctx context.Context, conversation string) error {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if !models.ValidID(conversation) {
		return &models.Error{Code: models.Validation}
	}
	owner, err := service.owner(ctx)
	if err != nil {
		return err
	}
	key, err := service.sessionKey(ctx, owner, conversation)
	if err != nil {
		return err
	}
	return service.records.SoftDelete(ctx, key)
}
