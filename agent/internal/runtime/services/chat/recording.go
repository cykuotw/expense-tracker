package chat

import (
	"context"
	"errors"
	"time"

	"expense-tracker/agent/internal/models"
	sessionstore "expense-tracker/agent/internal/store/session"
)

// runRecording tracks committed lifecycle events separately from their delivery.
// It belongs to one Run call; it does not own admission or ADK execution.
type runRecording struct {
	records           sessionstore.Repository
	run               models.Run
	emit              func(models.Output) error
	startCommitted    bool
	terminalCommitted bool
	deliveryFailed    bool
}

func (recording *runRecording) persistAndEmit(ctx context.Context, output models.Output) error {
	return recording.records.PersistAndEmitOutput(ctx, recording.run, output, func(stored models.Output) error {
		// The repository calls back only after the output has been committed.
		if stored.Kind == "started" {
			recording.startCommitted = true
		}
		if output.Kind == "finished" || output.Kind == "error" {
			recording.terminalCommitted = true
		}
		err := recording.emit(stored)
		if err != nil {
			recording.deliveryFailed = true
		}
		return err
	})
}

func (recording *runRecording) finalizeFailure(ctx context.Context, output models.Output, failure error) {
	if failure == nil || recording.terminalCommitted {
		return
	}
	cleanup, stop := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer stop()
	status, code := "failed", string(models.PublicCode(failure))
	if ctx.Err() != nil || errors.Is(failure, context.Canceled) || errors.Is(failure, context.DeadlineExceeded) {
		status, code = "interrupted", "interrupted"
	}
	output.Kind = "error"
	output.Delta = ""
	output.ErrorCode = code
	written := false
	err := recording.records.PersistAndEmitOutput(cleanup, recording.run, output, func(event models.Output) error {
		written = true
		if ctx.Err() != nil || recording.deliveryFailed {
			return nil
		}
		return recording.emit(event)
	})
	if err != nil && !written {
		_ = recording.records.Finish(cleanup, recording.run, status, code)
	}
}
