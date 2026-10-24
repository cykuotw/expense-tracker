package errornotifier

import (
	"context"
	"encoding/json/v2"
	"errors"
	"io"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/stretchr/testify/require"
)

type fakeAlertReader struct {
	body  string
	err   error
	calls int
}

func (f *fakeAlertReader) GetObject(_ context.Context, in *s3.GetObjectInput, _ ...func(*s3.Options)) (*s3.GetObjectOutput, error) {
	f.calls++
	return &s3.GetObjectOutput{Body: io.NopCloser(strings.NewReader(f.body))}, f.err
}

type channelPoster struct {
	urls     []string
	messages []string
}

func (p *channelPoster) Post(_ context.Context, u *url.URL, message string) error {
	p.urls = append(p.urls, u.String())
	p.messages = append(p.messages, message)
	return nil
}

func alertFixture(t *testing.T) (databaseAlert, events.S3Event) {
	t.Helper()
	alert := databaseAlert{Version: 1, EventID: "1b0685e9-aeea-4033-a2b5-e010c1879657", State: "ALARM", Connections: new(16), Maximum: new(20), Threshold: 80, ObservedAt: time.Now().UTC()}
	var record events.S3EventRecord
	record.EventSource = "aws:s3"
	record.EventName = "ObjectCreated:Put"
	record.S3.Bucket.Name = "private-backup"
	record.S3.Object.Key = "database-alerts/" + alert.EventID + ".json"
	record.S3.Object.Size = 300
	return alert, events.S3Event{Records: []events.S3EventRecord{record}}
}

func TestDatabaseNotificationsUseDedicatedChannelAndBoundedNumericPayload(t *testing.T) {
	alert, event := alertFixture(t)
	dbURL := strings.Replace(testWebhook, "123456789012345678", "223456789012345678", 1)
	webhook, err := NewWebhookProvider(dbURL)
	require.NoError(t, err)
	reader := &fakeAlertReader{}
	poster := &channelPoster{}
	handler, err := NewDatabaseHandler("private-backup", "production", reader, webhook, poster)
	require.NoError(t, err)
	for _, state := range []string{"ALARM", "OK", "CHECK_FAILED"} {
		alert.State = state
		if state == "OK" {
			alert.Connections = new(5)
		}
		if state == "CHECK_FAILED" {
			alert.Connections = nil
			alert.Maximum = nil
		}
		raw, err := json.Marshal(alert)
		require.NoError(t, err)
		reader.body = string(raw)
		require.NoError(t, handler.Handle(t.Context(), event))
	}
	require.Len(t, poster.urls, 3)
	for _, u := range poster.urls {
		require.Equal(t, dbURL, u)
		require.NotEqual(t, testWebhook, u)
	}
	require.Contains(t, poster.messages[0], "16/20 (80.0%)")
	require.Contains(t, poster.messages[1], "connection OK")
	require.Contains(t, poster.messages[2], "usage is unknown")
	for _, message := range poster.messages {
		require.NotContains(t, message, "discord.com")
		require.Less(t, len(message), maxDiscordMessageBytes)
	}
}

func TestDatabaseNotificationsRejectWrongObjectsAndPayloadWithoutLeakingDetails(t *testing.T) {
	alert, event := alertFixture(t)
	webhook, err := NewWebhookProvider(testWebhook)
	require.NoError(t, err)
	raw, err := json.Marshal(alert)
	require.NoError(t, err)
	reader := &fakeAlertReader{body: string(raw)}
	poster := &channelPoster{}
	handler, err := NewDatabaseHandler("private-backup", "production", reader, webhook, poster)
	require.NoError(t, err)
	for _, mutate := range []func(*events.S3Event){
		func(e *events.S3Event) { e.Records[0].S3.Bucket.Name = "other-bucket" },
		func(e *events.S3Event) { e.Records[0].S3.Object.Key = "daily/private.dump" },
		func(e *events.S3Event) { e.Records[0].S3.Object.Size = 4097 },
	} {
		_, event := alertFixture(t)
		mutate(&event)
		require.Error(t, handler.Handle(t.Context(), event))
		require.Zero(t, reader.calls)
	}
	for _, body := range []string{string(raw[:10]), strings.Replace(string(raw), `"connections":16`, `"connections":2`, 1), strings.Replace(string(raw), `"version":1`, `"version":1,"query":"private"`, 1), strings.Repeat("x", 4097)} {
		reader.body = body
		require.Error(t, handler.Handle(t.Context(), event))
		require.Empty(t, poster.messages)
	}
	reader.err = errors.New("private endpoint detail")
	err = handler.Handle(t.Context(), event)
	require.True(t, IsRetryable(err))
	require.NotContains(t, err.Error(), "private endpoint detail")
}
