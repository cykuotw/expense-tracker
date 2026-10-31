package chat

import (
	"context"
	"time"
	"uuid"

	"expense-tracker/agent/internal/llm"
	"expense-tracker/agent/internal/models"
	"expense-tracker/agent/internal/runtime/engine"
	sessionstore "expense-tracker/agent/internal/store/session"
)

const runTimeout = 60 * time.Second

// Run validates and admits one message, then executes it with optional recording.
func (service *Service) Run(parent context.Context, request models.ChatRequest, emit func(models.Output) error) (failure error) {
	if err := parent.Err(); err != nil {
		return err
	}
	if err := request.Validate(); err != nil {
		return err
	}
	owner, err := service.resolveOwner(parent)
	if err != nil {
		return err
	}
	if request.RunID == "" {
		request.RunID = uuid.New().String()
	}
	key, err := service.reserveRun(owner, request)
	if err != nil {
		return err
	}

	var recording *runRecording
	defer func() {
		allowRetry := service.records != nil && (recording == nil || !recording.startCommitted)
		service.releaseRun(owner, request, allowRetry)
	}()
	ctx, cancel := context.WithTimeout(llm.WithRunBudget(parent, service.limits), runTimeout)
	defer cancel()
	output := models.Output{
		Kind:            "started",
		RunID:           request.RunID,
		ConversationID:  request.ConversationID,
		ClientMessageID: request.MessageID,
	}
	publish := emit
	if service.records != nil {
		key, err = service.records.ResolveThread(ctx, engine.AppName, owner, request.ConversationID, true)
		if err != nil {
			return err
		}
		run, err := service.records.BeginUI(ctx, key,
			models.UIRef{ThreadID: request.ConversationID, RunID: request.RunID},
			request.MessageID, request)
		if err != nil {
			return err
		}
		ctx = sessionstore.WithRun(ctx, service.records, run)
		output.MessageID = run.MessageID
		recording = &runRecording{records: service.records, run: run, emit: emit}
		publish = func(event models.Output) error { return recording.persistAndEmit(ctx, event) }
		defer func() { recording.finalizeFailure(ctx, output, failure) }()
	}

	if err := publish(output); err != nil {
		return err
	}
	if err := service.engine.RunTurn(ctx, key, request, output, publish); err != nil {
		return err
	}
	output.Kind = "finished"
	return publish(output)
}
