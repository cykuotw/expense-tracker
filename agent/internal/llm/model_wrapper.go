package llm

import (
	"context"
	"encoding/json/v2"
	"iter"
	"time"
	"uuid"

	"expense-tracker/agent/internal/models"
	sessionstore "expense-tracker/agent/internal/store/session"

	"google.golang.org/adk/v2/agent"
	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

type modelWrapper struct {
	limits          models.LLMLimits
	maxOutputTokens int32
	model.LLM
	skillIdentity []string
}

// WithLimitsAndRecording wraps a model with turn and payload limits and
// diagnostic recording. Construction performs no inference.
func WithLimitsAndRecording(provider model.LLM, limits models.LLMLimits, maxOutputTokens int32, skillIdentity ...string) model.LLM {
	return &modelWrapper{LLM: provider, limits: limits, maxOutputTokens: maxOutputTokens, skillIdentity: skillIdentity}
}

func (wrappedModel *modelWrapper) GenerateContent(
	ctx context.Context,
	request *model.LLMRequest,
	stream bool,
) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		if err := consumeModelCall(ctx); err != nil {
			yield(nil, err)
			return
		}
		if err := applyRequestLimits(request, wrappedModel.maxOutputTokens, wrappedModel.limits.MaxContextBytes); err != nil {
			yield(nil, err)
			return
		}
		callID := uuid.New().String()
		invocation, author, branch := "", "", ""
		if agentContext, ok := ctx.(agent.ReadonlyContext); ok {
			invocation = agentContext.InvocationID()
			author = agentContext.AgentName()
			branch = agentContext.Branch()
		}
		metadata := map[string]any{"callId": callID, "agent": author, "branch": branch, "skill": wrappedModel.skillIdentity, "model": request.Model}
		metadata["request"] = struct {
			Model    string
			Contents []*genai.Content
			Config   *genai.GenerateContentConfig
		}{request.Model, request.Contents, request.Config}
		if err := sessionstore.RecordDiagnostic(ctx, "model.request", invocation, metadata); err != nil {
			yield(nil, err)
			return
		}
		delete(metadata, "request")
		started := time.Now()
		status := "interrupted"
		defer func() {
			if status == "completed" {
				return
			}
			metadata["durationMs"] = time.Since(started).Milliseconds()
			metadata["status"] = status
			cleanup, cancel := context.WithTimeout(context.WithoutCancel(ctx), 2*time.Second)
			defer cancel()
			_ = sessionstore.RecordDiagnostic(cleanup, "model.finished", invocation, metadata)
		}()

		responses := responseUsage{limits: wrappedModel.limits}
		for response, err := range wrappedModel.LLM.GenerateContent(ctx, request, stream) {
			if err != nil {
				status = "provider_error"
			}
			if err == nil && response != nil {
				if err := responses.record(response); err != nil {
					_ = sessionstore.RecordDiagnostic(ctx, "model.response", invocation, map[string]any{"callId": callID, "response": response, "rejected": "model_response_limit", "reasoningAvailable": hasThoughtParts(response)})
					yield(nil, err)
					return
				}
				if err := sessionstore.RecordDiagnostic(ctx, "model.response", invocation, map[string]any{"callId": callID, "response": response, "reasoningAvailable": hasThoughtParts(response)}); err != nil {
					yield(nil, err)
					return
				}
				response = sanitizeResponse(response)
			}
			if !yield(response, err) || err != nil {
				return
			}
		}
		status = "completed"
		metadata["durationMs"] = time.Since(started).Milliseconds()
		metadata["status"] = status
		if err := sessionstore.RecordDiagnostic(ctx, "model.finished", invocation, metadata); err != nil {
			yield(nil, err)
		}
	}
}

// Function arguments/results count toward the context limit as serialized JSON.
func applyRequestLimits(request *model.LLMRequest, limit, contextLimit int32) error {
	if limit <= 0 || contextLimit <= 0 {
		return models.ErrBudget
	}
	if request.Config == nil {
		request.Config = new(genai.GenerateContentConfig)
	}
	if request.Config.MaxOutputTokens <= 0 || request.Config.MaxOutputTokens > limit {
		request.Config.MaxOutputTokens = limit
	}

	contextBytes := 0
	for _, content := range request.Contents {
		if content == nil {
			continue
		}
		payload, err := json.Marshal(content)
		if err != nil {
			return models.ErrBudget
		}
		contextBytes += len(payload)
	}
	if contextBytes > int(contextLimit) {
		return models.ErrBudget
	}
	return nil
}
