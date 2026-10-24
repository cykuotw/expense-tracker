package models

import (
	"context"
	"encoding/json/v2"
	"errors"
	"iter"
	"slices"
	"sync/atomic"

	"google.golang.org/adk/v2/model"
	"google.golang.org/genai"
)

const (
	maxModelCalls         = 6
	maxToolCalls          = 2
	maxOutputTokens       = 2048
	maxContextBytes       = 64 << 10
	maxResponseBytes      = 32 << 10
	maxTotalResponseBytes = 256 << 10
	maxResponseEvents     = 256
)

var ErrBudget = errors.New("execution budget exceeded")

type Budget struct {
	Models atomic.Int32
	Tools  atomic.Int32
}

type budgetKey struct{}

func WithBudget(ctx context.Context) context.Context {
	return context.WithValue(ctx, budgetKey{}, new(Budget))
}

func TakeTool(ctx context.Context) error {
	budget, exists := ctx.Value(budgetKey{}).(*Budget)
	if !exists || budget.Tools.Add(1) > maxToolCalls {
		return ErrBudget
	}
	return ctx.Err()
}

type limited struct {
	model.LLM
}

func Limited(provider model.LLM) model.LLM {
	return &limited{LLM: provider}
}

func (limitedModel *limited) GenerateContent(
	ctx context.Context,
	request *model.LLMRequest,
	stream bool,
) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		budget, exists := ctx.Value(budgetKey{}).(*Budget)
		if !exists || budget.Models.Add(1) > maxModelCalls {
			yield(nil, ErrBudget)
			return
		}
		if err := ctx.Err(); err != nil {
			yield(nil, err)
			return
		}
		if err := prepareRequest(request); err != nil {
			yield(nil, err)
			return
		}

		var responses responseBudget
		for response, err := range limitedModel.LLM.GenerateContent(ctx, request, stream) {
			if err == nil && response != nil {
				if err := responses.check(response); err != nil {
					yield(nil, err)
					return
				}
				response = withoutPrivateMetadata(response)
			}
			if !yield(response, err) || err != nil {
				return
			}
		}
	}
}

// Function arguments/results count toward the context limit as serialized JSON.
func prepareRequest(request *model.LLMRequest) error {
	if request.Config == nil {
		request.Config = new(genai.GenerateContentConfig)
	}
	request.Config.MaxOutputTokens = maxOutputTokens

	contextBytes := 0
	for _, content := range request.Contents {
		if content == nil {
			continue
		}
		payload, err := json.Marshal(content)
		if err != nil {
			return ErrBudget
		}
		contextBytes += len(payload)
	}
	if contextBytes > maxContextBytes {
		return ErrBudget
	}
	return nil
}

type responseBudget struct {
	events int
	bytes  int
}

func (budget *responseBudget) check(response *model.LLMResponse) error {
	payload, err := json.Marshal(response)
	budget.events++
	budget.bytes += len(payload)

	overLimit := len(payload) > maxResponseBytes ||
		budget.bytes > maxTotalResponseBytes ||
		budget.events > maxResponseEvents
	if err != nil || overLimit {
		return ErrBudget
	}
	return nil
}

func withoutPrivateMetadata(response *model.LLMResponse) *model.LLMResponse {
	sanitized := *response
	sanitized.CustomMetadata = nil
	if response.Content != nil {
		content := *response.Content
		content.Parts = slices.DeleteFunc(
			slices.Clone(content.Parts),
			func(part *genai.Part) bool {
				return part == nil || part.Thought
			},
		)
		sanitized.Content = &content
	}
	return &sanitized
}
