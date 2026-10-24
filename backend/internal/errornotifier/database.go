package errornotifier

import (
	"context"
	"encoding/json/v2"
	"fmt"
	"io"
	"net/url"
	"regexp"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/s3"
)

var databaseAlertKey = regexp.MustCompile(`^database-alerts/[0-9a-f-]{36}\.json$`)

type AlertObjectReader interface {
	GetObject(context.Context, *s3.GetObjectInput, ...func(*s3.Options)) (*s3.GetObjectOutput, error)
}

type DatabaseHandler struct {
	bucket      string
	environment string
	reader      AlertObjectReader
	webhook     webhookSource
	discord     discordPoster
}

func NewDatabaseHandler(bucket, environment string, reader AlertObjectReader, webhook webhookSource, discord discordPoster) (*DatabaseHandler, error) {
	if bucket == "" || !validRuntimeLabel(environment) || reader == nil || webhook == nil || discord == nil {
		return nil, permanent("configure database alert handler")
	}
	return &DatabaseHandler{bucket, environment, reader, webhook, discord}, nil
}

type databaseAlert struct {
	Version     int       `json:"version"`
	EventID     string    `json:"event_id"`
	State       string    `json:"state"`
	Connections *int      `json:"connections"`
	Maximum     *int      `json:"max_connections"`
	Threshold   int       `json:"threshold_percent"`
	ObservedAt  time.Time `json:"observed_at"`
}

func (h *DatabaseHandler) Handle(ctx context.Context, event events.S3Event) error {
	if len(event.Records) == 0 || len(event.Records) > 5 {
		return permanent("validate database alert records")
	}
	for _, record := range event.Records {
		key, err := url.QueryUnescape(record.S3.Object.Key)
		if err != nil || record.EventSource != "aws:s3" || record.S3.Bucket.Name != h.bucket ||
			(record.EventName != "ObjectCreated:Put" && record.EventName != "ObjectCreated:CompleteMultipartUpload") ||
			!databaseAlertKey.MatchString(key) || record.S3.Object.Size < 1 || record.S3.Object.Size > 4096 {
			return permanent("validate database alert object")
		}
		result, err := h.reader.GetObject(ctx, &s3.GetObjectInput{Bucket: aws.String(h.bucket), Key: aws.String(key)})
		if err != nil {
			return retryable("read database alert object")
		}
		if result == nil || result.Body == nil {
			return retryable("read database alert body")
		}
		raw, readErr := io.ReadAll(io.LimitReader(result.Body, 4097))
		result.Body.Close()
		if readErr != nil {
			return retryable("read database alert body")
		}
		var alert databaseAlert
		if len(raw) > 4096 || json.Unmarshal(raw, &alert, json.RejectUnknownMembers(true)) != nil || !alert.valid(key) {
			return permanent("validate database alert payload")
		}
		webhook, err := h.webhook.URL(ctx)
		if err != nil {
			return err
		}
		if err := h.discord.Post(ctx, webhook, h.message(alert)); err != nil {
			return err
		}
	}
	return nil
}

func (a databaseAlert) valid(key string) bool {
	if a.Version != 1 || !uuidPattern.MatchString(a.EventID) || key != "database-alerts/"+a.EventID+".json" ||
		a.Threshold != 80 || a.ObservedAt.IsZero() || a.ObservedAt.After(time.Now().Add(5*time.Minute)) {
		return false
	}
	if a.State == "CHECK_FAILED" {
		return a.Connections == nil && a.Maximum == nil
	}
	if a.Connections == nil || a.Maximum == nil || *a.Connections < 0 || *a.Maximum < 1 || *a.Maximum > 1_000_000 || *a.Connections >= *a.Maximum {
		return false
	}
	high := *a.Connections*100 >= *a.Maximum*a.Threshold
	return a.State == "ALARM" && high || a.State == "OK" && !high
}

func (h *DatabaseHandler) message(a databaseAlert) string {
	detail := "PostgreSQL connection checks failed twice; connection usage is unknown."
	if a.State != "CHECK_FAILED" {
		detail = fmt.Sprintf("PostgreSQL connections: %d/%d (%.1f%%), threshold %d%%. Two consecutive samples.",
			*a.Connections, *a.Maximum, float64(*a.Connections)*100/float64(*a.Maximum), a.Threshold)
	}
	return fmt.Sprintf("[%s] database connection %s\n%s\nObserved: %s\nEvent: %s\nDelivery is at-least-once; delayed or duplicate notifications are possible.",
		h.environment, a.State, detail, a.ObservedAt.UTC().Format(time.RFC3339), a.EventID)
}
