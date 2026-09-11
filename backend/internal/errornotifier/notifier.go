package errornotifier

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

const (
	dataMessage               = "DATA_MESSAGE"
	controlMessage            = "CONTROL_MESSAGE"
	maxInputEvents            = 200
	maxLogMessageBytes        = 16 * 1024
	maxDiscordMessageBytes    = 1800
	maxDiscordSamples         = 8
	maxMessageFooterBytes     = 160
	maxRouteBytes             = 160
	maxCodeBytes              = 64
	maxCorrelationIDBytes     = 128
	maxCloudWatchEventIDBytes = 128
	maxRuntimeLabelBytes      = 64
)

var (
	runtimeLabelPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*$`)
	routePattern        = regexp.MustCompile(`^(?:unmatched|/[A-Za-z0-9_./:{}*-]*)$`)
	codePattern         = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	correlationPattern  = regexp.MustCompile(`^[A-Za-z0-9_.:/+=-]+$`)
)

type webhookSource interface {
	URL(context.Context) (*url.URL, error)
}

type discordPoster interface {
	Post(context.Context, *url.URL, string) error
}

type Handler struct {
	environment string
	function    string
	webhook     webhookSource
	discord     discordPoster
}

func NewHandler(environment, function string, webhook webhookSource, discord discordPoster) (*Handler, error) {
	if !validRuntimeLabel(environment) || !validRuntimeLabel(function) || webhook == nil || discord == nil {
		return nil, permanent("configure handler")
	}
	return &Handler{environment: environment, function: function, webhook: webhook, discord: discord}, nil
}

func (h *Handler) Handle(ctx context.Context, event events.CloudwatchLogsEvent) error {
	data, err := decodeSubscription(event)
	if err != nil {
		return err
	}
	switch data.MessageType {
	case controlMessage:
		return nil
	case dataMessage:
	default:
		return permanent("validate subscription message type")
	}

	alerts, skipped := selectAlerts(data.LogEvents)
	if len(alerts) == 0 {
		return nil
	}
	message := h.message(alerts, skipped)
	webhook, err := h.webhook.URL(ctx)
	if err != nil {
		return err
	}
	return h.discord.Post(ctx, webhook, message)
}

type alert struct {
	eventID    string
	timestamp  time.Time
	event      string
	route      string
	status     int
	errorCode  string
	requestID  string
	apiRequest string
	awsRequest string
}

type logAlert struct {
	Event               string `json:"event"`
	Alertable           bool   `json:"alertable"`
	Route               string `json:"route"`
	Status              int    `json:"status"`
	ErrorCode           string `json:"error_code"`
	RequestID           string `json:"request_id"`
	APIGatewayRequestID string `json:"api_gateway_request_id"`
	AWSRequestID        string `json:"aws_request_id"`
}

func selectAlerts(events []events.CloudwatchLogsLogEvent) ([]alert, int) {
	limit := min(len(events), maxInputEvents)
	alerts := make([]alert, 0, min(limit, maxDiscordSamples))
	for i := range limit {
		if parsed, ok := parseAlert(events[i]); ok {
			alerts = append(alerts, parsed)
		}
	}
	slices.SortFunc(alerts, func(left, right alert) int {
		if compared := left.timestamp.Compare(right.timestamp); compared != 0 {
			return compared
		}
		return strings.Compare(left.eventID, right.eventID)
	})
	return alerts, len(events) - limit
}

func parseAlert(event events.CloudwatchLogsLogEvent) (alert, bool) {
	if len(event.Message) == 0 || len(event.Message) > maxLogMessageBytes || event.Timestamp <= 0 ||
		!validSafeField(event.ID, maxCloudWatchEventIDBytes, correlationPattern) {
		return alert{}, false
	}
	var value logAlert
	if err := json.Unmarshal([]byte(event.Message), &value); err != nil || !value.Alertable {
		return alert{}, false
	}
	if value.Event != "unexpected_http_error" && value.Event != "panic_recovered" {
		return alert{}, false
	}
	if value.Status < 500 || value.Status > 599 ||
		!validSafeField(value.Route, maxRouteBytes, routePattern) ||
		!validSafeField(value.ErrorCode, maxCodeBytes, codePattern) {
		return alert{}, false
	}
	for _, identifier := range []string{value.RequestID, value.APIGatewayRequestID, value.AWSRequestID} {
		if identifier != "" && !validSafeField(identifier, maxCorrelationIDBytes, correlationPattern) {
			return alert{}, false
		}
	}
	return alert{
		eventID: event.ID, timestamp: time.UnixMilli(event.Timestamp).UTC(), event: value.Event,
		route: value.Route, status: value.Status, errorCode: value.ErrorCode,
		requestID: value.RequestID, apiRequest: value.APIGatewayRequestID, awsRequest: value.AWSRequestID,
	}, true
}

func (h *Handler) message(alerts []alert, skipped int) string {
	var message strings.Builder
	fmt.Fprintf(&message, "[%s] %s: %d backend alert(s)\n", h.environment, h.function, len(alerts))
	displayed := 0
	for _, item := range alerts[:min(len(alerts), maxDiscordSamples)] {
		line := formatAlert(item)
		if message.Len()+len(line)+maxMessageFooterBytes > maxDiscordMessageBytes {
			break
		}
		message.WriteString(line)
		displayed++
	}
	suppressed := skipped + len(alerts) - displayed
	if suppressed > 0 {
		fmt.Fprintf(&message, "Suppressed input/sample count: %d\n", suppressed)
	}
	message.WriteString("CloudWatch delivery is at-least-once; duplicate posts are possible.")
	return message.String()
}

func formatAlert(value alert) string {
	summary := "unexpected server failure"
	if value.event == "panic_recovered" {
		summary = "recovered application panic"
	}
	lookup := "lookup=event_id:" + value.eventID
	switch {
	case value.requestID != "":
		lookup = "request_id=" + value.requestID
	case value.awsRequest != "":
		lookup = "aws_request_id=" + value.awsRequest
	case value.apiRequest != "":
		lookup = "api_gateway_request_id=" + value.apiRequest
	}
	return fmt.Sprintf("- %s %s route=%s status=%d code=%s %s event_id=%s\n",
		value.timestamp.Format(time.RFC3339), summary, value.route, value.status,
		value.errorCode, lookup, value.eventID,
	)
}

func validRuntimeLabel(value string) bool {
	return validSafeField(value, maxRuntimeLabelBytes, runtimeLabelPattern)
}

func validSafeField(value string, maximum int, pattern *regexp.Regexp) bool {
	return len(value) > 0 && len(value) <= maximum && pattern.MatchString(value)
}
