package errornotifier

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/stretchr/testify/require"
)

const testWebhook = "https://discord.com/api/webhooks/123456789012345678/abcdefghijklmnopqrstuvwxyz_ABCD"

type fakeWebhookSource struct {
	url   *url.URL
	err   error
	calls int
}

func (s *fakeWebhookSource) URL(context.Context) (*url.URL, error) {
	s.calls++
	return s.url, s.err
}

type fakeDiscordPoster struct {
	message string
	calls   int
	err     error
}

func (p *fakeDiscordPoster) Post(_ context.Context, _ *url.URL, message string) error {
	p.calls++
	p.message = message
	return p.err
}

func subscriptionEvent(t *testing.T, data events.CloudwatchLogsData) events.CloudwatchLogsEvent {
	t.Helper()
	value, err := json.Marshal(data)
	require.NoError(t, err)
	return compressedEvent(t, value)
}

func compressedEvent(t *testing.T, value []byte) events.CloudwatchLogsEvent {
	t.Helper()
	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	_, err := writer.Write(value)
	require.NoError(t, err)
	require.NoError(t, writer.Close())
	return events.CloudwatchLogsEvent{AWSLogs: events.CloudwatchLogsRawData{
		Data: base64.StdEncoding.EncodeToString(compressed.Bytes()),
	}}
}

func canonicalLogEvent(t *testing.T, index int, eventType string) events.CloudwatchLogsLogEvent {
	t.Helper()
	message, err := json.Marshal(map[string]any{
		"timestamp":               "2026-09-11T12:00:00Z",
		"level":                   "ERROR",
		"event":                   eventType,
		"alertable":               true,
		"route":                   "/api/v0/groups/:id",
		"status":                  500,
		"error_code":              "internal_error",
		"error_category":          "database_operation",
		"error_type":              "*errors.errorString",
		"diagnostic_message":      "CANARY_DIAGNOSTIC_SECRET",
		"stack":                   "CANARY_STACK_SECRET",
		"request_id":              fmt.Sprintf("request-%03d", index),
		"api_gateway_request_id":  fmt.Sprintf("gateway-%03d", index),
		"aws_request_id":          fmt.Sprintf("lambda-%03d", index),
		"unknown_sensitive_field": "CANARY_UNKNOWN_SECRET",
	})
	require.NoError(t, err)
	return events.CloudwatchLogsLogEvent{
		ID:        fmt.Sprintf("event-%03d", index),
		Timestamp: 1_757_592_000_000 + int64(index),
		Message:   string(message),
	}
}

func TestHandlerSendsBoundedSanitizedDeterministicMessage(t *testing.T) {
	webhookURL, err := url.Parse(testWebhook)
	require.NoError(t, err)
	webhook := &fakeWebhookSource{url: webhookURL}
	discord := &fakeDiscordPoster{}
	handler, err := NewHandler("production", "expense-worker-errors", webhook, discord)
	require.NoError(t, err)

	logEvents := make([]events.CloudwatchLogsLogEvent, 0, 12)
	for i := range 12 {
		eventType := "unexpected_http_error"
		if i%2 == 0 {
			eventType = "panic_recovered"
		}
		logEvents = append(logEvents, canonicalLogEvent(t, 11-i, eventType))
	}
	err = handler.Handle(t.Context(), subscriptionEvent(t, events.CloudwatchLogsData{
		MessageType: dataMessage,
		LogEvents:   logEvents,
	}))
	require.NoError(t, err)
	require.Equal(t, 1, webhook.calls)
	require.Equal(t, 1, discord.calls)
	require.LessOrEqual(t, len(discord.message), maxDiscordMessageBytes)
	require.Contains(t, discord.message, "[production] expense-worker-errors: 12 backend alert(s)")
	require.Contains(t, discord.message, "request_id=request-000")
	require.Contains(t, discord.message, "Suppressed input/sample count: 4")
	require.Less(t, strings.Index(discord.message, "event-000"), strings.Index(discord.message, "event-001"))
	for _, prohibited := range []string{
		"CANARY_DIAGNOSTIC_SECRET",
		"CANARY_STACK_SECRET",
		"CANARY_UNKNOWN_SECRET",
		"error_type",
	} {
		require.NotContains(t, discord.message, prohibited)
	}
}

func TestHandlerIgnoresControlAndNoncanonicalEvents(t *testing.T) {
	webhook := &fakeWebhookSource{}
	discord := &fakeDiscordPoster{}
	handler, err := NewHandler("production", "expense-worker-errors", webhook, discord)
	require.NoError(t, err)

	require.NoError(t, handler.Handle(t.Context(), subscriptionEvent(t, events.CloudwatchLogsData{
		MessageType: controlMessage,
	})))
	noncanonical := canonicalLogEvent(t, 1, "request_completed")
	require.NoError(t, handler.Handle(t.Context(), subscriptionEvent(t, events.CloudwatchLogsData{
		MessageType: dataMessage,
		LogEvents: []events.CloudwatchLogsLogEvent{
			noncanonical,
			{ID: "malformed", Timestamp: 1, Message: "not-json CANARY_RAW_CONTENT"},
		},
	})))
	require.Zero(t, webhook.calls)
	require.Zero(t, discord.calls)
}

func TestHandlerClassifiesMalformedAndUnsupportedPayloadsWithoutRawContent(t *testing.T) {
	handler, err := NewHandler(
		"production",
		"expense-worker-errors",
		&fakeWebhookSource{},
		&fakeDiscordPoster{},
	)
	require.NoError(t, err)

	for name, event := range map[string]events.CloudwatchLogsEvent{
		"invalid base64": {
			AWSLogs: events.CloudwatchLogsRawData{Data: "CANARY_RAW_CONTENT"},
		},
		"unsupported type": subscriptionEvent(t, events.CloudwatchLogsData{
			MessageType: "CANARY_RAW_CONTENT",
		}),
		"invalid gzip": {
			AWSLogs: events.CloudwatchLogsRawData{
				Data: base64.StdEncoding.EncodeToString([]byte("CANARY_RAW_CONTENT")),
			},
		},
	} {
		t.Run(name, func(t *testing.T) {
			err := handler.Handle(t.Context(), event)
			require.Error(t, err)
			require.False(t, IsRetryable(err))
			require.NotContains(t, err.Error(), "CANARY_RAW_CONTENT")
		})
	}
}

func TestAlertSelectionRequiresCanonicalSafeFields(t *testing.T) {
	valid := canonicalLogEvent(t, 1, "panic_recovered")
	cases := []map[string]any{
		{"event": "request_completed", "alertable": true, "route": "/api/v0/groups", "status": 500, "error_code": "internal_error"},
		{"event": "unexpected_http_error", "alertable": false, "route": "/api/v0/groups", "status": 500, "error_code": "internal_error"},
		{"event": "unexpected_http_error", "alertable": true, "route": "/api/v0/groups", "status": 400, "error_code": "bad_request"},
		{"event": "unexpected_http_error", "alertable": true, "route": "/api/@everyone", "status": 500, "error_code": "internal_error"},
		{"event": "unexpected_http_error", "alertable": true, "route": "/api/v0/groups", "status": 500, "error_code": "Unsafe-Code"},
	}
	logEvents := []events.CloudwatchLogsLogEvent{valid}
	for index, value := range cases {
		message, err := json.Marshal(value)
		require.NoError(t, err)
		logEvents = append(logEvents, events.CloudwatchLogsLogEvent{
			ID:        fmt.Sprintf("rejected-%d", index),
			Timestamp: 1_757_592_000_000,
			Message:   string(message),
		})
	}
	alerts, skipped := selectAlerts(logEvents)
	require.Zero(t, skipped)
	require.Len(t, alerts, 1)
	require.Equal(t, "event-001", alerts[0].eventID)
}

func TestAlertLookupUsesOnlyBoundedCorrelationFields(t *testing.T) {
	base := alert{
		eventID:   "event-id",
		timestamp: time.Unix(0, 0).UTC(),
		event:     "unexpected_http_error",
		route:     "/api/v0/groups",
		status:    500,
		errorCode: "internal_error",
	}
	base.apiRequest = "gateway-id"
	require.Contains(t, formatAlert(base), "api_gateway_request_id=gateway-id")
	base.awsRequest = "lambda-id"
	require.Contains(t, formatAlert(base), "aws_request_id=lambda-id")
	base.requestID = "request-id"
	require.Contains(t, formatAlert(base), "request_id=request-id")
}

func TestDecodeRejectsOversizedAndDeepPayloads(t *testing.T) {
	oversized := subscriptionEvent(t, events.CloudwatchLogsData{
		MessageType: dataMessage,
		LogEvents: []events.CloudwatchLogsLogEvent{{
			ID: "event", Timestamp: 1, Message: strings.Repeat("x", maxDecompressedPayloadBytes),
		}},
	})
	_, err := decodeSubscription(oversized)
	require.Error(t, err)
	require.False(t, IsRetryable(err))

	deep := strings.Repeat("[", maxJSONDepth+1) + strings.Repeat("]", maxJSONDepth+1)
	_, err = decodeSubscription(compressedEvent(t, []byte(deep)))
	require.Error(t, err)
	require.False(t, IsRetryable(err))
}

func TestInputEventLimitIsReported(t *testing.T) {
	webhookURL, err := url.Parse(testWebhook)
	require.NoError(t, err)
	discord := &fakeDiscordPoster{}
	handler, err := NewHandler(
		"production",
		"expense-worker-errors",
		&fakeWebhookSource{url: webhookURL},
		discord,
	)
	require.NoError(t, err)
	logEvents := make([]events.CloudwatchLogsLogEvent, maxInputEvents+3)
	for i := range logEvents {
		logEvents[i] = canonicalLogEvent(t, i, "unexpected_http_error")
	}
	require.NoError(t, handler.Handle(t.Context(), subscriptionEvent(t, events.CloudwatchLogsData{
		MessageType: dataMessage,
		LogEvents:   logEvents,
	})))
	require.Contains(t, discord.message, "Suppressed input/sample count: 195")
}
