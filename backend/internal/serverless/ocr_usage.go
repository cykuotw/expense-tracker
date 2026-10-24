package serverless

import (
	"context"
	"encoding/json"
	jsonv2 "encoding/json/v2"
	"errors"
	"time"

	"expense-tracker/backend/services/ocrusage"
	"github.com/aws/aws-lambda-go/events"
)

type UsageRecorder interface {
	Save(context.Context, ocrusage.Record) error
}
type GatewayProxy func(context.Context, events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error)

// HandleWorkerEvent accepts usage only in a direct IAM Lambda event. API Gateway
// wraps browser content in body; that body is never interpreted as a usage event.
func HandleWorkerEvent(ctx context.Context, raw json.RawMessage, store UsageRecorder, proxy GatewayProxy) (any, error) {
	var envelope struct {
		Type    string `json:"type"`
		Version string `json:"version"`
	}
	if err := jsonv2.Unmarshal(raw, &envelope); err != nil {
		return nil, errors.New("invalid worker event")
	}
	if envelope.Type == ocrusage.EventType && envelope.Version == "" {
		var event ocrusage.Event
		if err := jsonv2.Unmarshal(raw, &event, jsonv2.RejectUnknownMembers(true)); err != nil {
			return nil, errors.New("invalid OCR usage event")
		}
		if err := event.Usage.Validate(); err != nil {
			return nil, err
		}
		ctx, cancel := context.WithTimeout(ctx, 3*time.Second)
		defer cancel()
		if err := store.Save(ctx, event.Usage); err != nil {
			return nil, errors.New("OCR usage persistence failed")
		}
		return struct {
			Saved bool `json:"saved"`
		}{true}, nil
	}
	if envelope.Type != "" || envelope.Version != "2.0" {
		return nil, errors.New("unsupported worker event")
	}
	var request events.APIGatewayV2HTTPRequest
	if err := jsonv2.Unmarshal(raw, &request); err != nil {
		return nil, errors.New("invalid gateway event")
	}
	return proxy(ctx, request)
}
