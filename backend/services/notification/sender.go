package notification

import (
	"context"
	"encoding/json"
	"errors"
	"expense-tracker/backend/types"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
)

const (
	senderBatchSize = 10
	senderTimeout   = 5 * time.Second
	maxAttempts     = 3
)

type DeliveryStore interface {
	ClaimDeliveries(context.Context) (DeliveryBatch, error)
	AcknowledgeDeliveries(context.Context, uuid.UUID, []DeliveryResult) error
}

type Sender struct {
	store      DeliveryStore
	publicKey  string
	privateKey string
	subscriber string
	client     *http.Client
	clock      func() time.Time
}

func NewSender(store DeliveryStore, publicKey, privateKey, subscriber string) (*Sender, error) {
	if strings.TrimSpace(publicKey) == "" || strings.TrimSpace(privateKey) == "" || !strings.HasPrefix(subscriber, "mailto:") {
		return nil, errors.New("WEB_PUSH_VAPID_PUBLIC_KEY, WEB_PUSH_VAPID_PRIVATE_KEY, and WEB_PUSH_VAPID_SUBJECT are required")
	}
	return &Sender{
		store:      store,
		publicKey:  publicKey,
		privateKey: privateKey,
		// webpush-go adds the mailto: scheme for non-HTTPS subjects.
		subscriber: strings.TrimPrefix(subscriber, "mailto:"),
		client: &http.Client{
			Timeout: senderTimeout,
			Transport: &http.Transport{
				DialContext:       publicPushDialer,
				ForceAttemptHTTP2: true,
				MaxIdleConns:      2,
				MaxConnsPerHost:   1,
				IdleConnTimeout:   15 * time.Second,
			},
			CheckRedirect: func(*http.Request, []*http.Request) error {
				return http.ErrUseLastResponse
			},
		},
		clock: time.Now,
	}, nil
}

func (s *Sender) RunOnce(ctx context.Context) error {
	// Reserve time to acknowledge completed sends before the Lambda timeout.
	sendCtx, cancel := context.WithTimeout(ctx, 45*time.Second)
	defer cancel()
	batch, err := s.store.ClaimDeliveries(sendCtx)
	if err != nil {
		return fmt.Errorf("claim deliveries: %w", err)
	}
	var sendErr error
	results := make([]DeliveryResult, 0, len(batch.Deliveries))
	for _, delivery := range batch.Deliveries {
		if sendErr = sendCtx.Err(); sendErr != nil {
			break
		}
		var result DeliveryResult
		result, sendErr = s.send(sendCtx, delivery)
		if sendErr != nil {
			break
		}
		results = append(results, result)
	}
	if len(results) == 0 {
		return sendErr
	}
	ackCtx, ackCancel := context.WithTimeout(ctx, 10*time.Second)
	defer ackCancel()
	return errors.Join(sendErr, s.store.AcknowledgeDeliveries(ackCtx, batch.Token, results))
}

func (s *Sender) send(ctx context.Context, delivery types.WebPushDelivery) (DeliveryResult, error) {
	result := DeliveryResult{ID: delivery.ID}
	payload, err := notificationPayload(delivery)
	if err != nil {
		return result, fmt.Errorf("marshal notification payload: %w", err)
	}
	ttl := max(1, int(time.Until(delivery.ExpiresAt).Seconds()))
	response, sendErr := webpush.SendNotificationWithContext(ctx, payload, &webpush.Subscription{
		Endpoint: delivery.Subscription.Endpoint,
		Keys:     webpush.Keys{Auth: delivery.Subscription.Auth, P256dh: delivery.Subscription.P256DH},
	}, &webpush.Options{
		HTTPClient:      s.client,
		Subscriber:      s.subscriber,
		VAPIDPublicKey:  s.publicKey,
		VAPIDPrivateKey: s.privateKey,
		TTL:             ttl,
		Topic:           "expense-tracker-activity",
	})
	if response != nil {
		defer response.Body.Close()
	}
	if sendErr == nil && response != nil && response.StatusCode >= http.StatusOK && response.StatusCode < http.StatusMultipleChoices {
		result.Status = "delivered"
		return result, nil
	}
	if response != nil && (response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone) {
		result.Status = "retired"
		return result, nil
	}
	if response != nil && response.StatusCode >= http.StatusBadRequest && response.StatusCode < http.StatusInternalServerError && response.StatusCode != http.StatusTooManyRequests {
		slog.Warn("web push provider rejected delivery",
			"delivery_id", delivery.ID,
			"subscription_id", delivery.Subscription.ID,
			"provider", pushProviderHost(delivery.Subscription.Endpoint),
			"status_code", response.StatusCode,
		)
		result.Status = "provider_rejected"
		return result, nil
	}

	attempts := delivery.Attempts + 1
	next := s.clock().UTC().Add(retryDelay(response))
	if attempts >= maxAttempts || !next.Before(delivery.ExpiresAt) {
		result.Status = "failed"
		return result, nil
	}

	if sendErr != nil {
		slog.Warn("web push delivery deferred", "attempt", attempts)
	}
	result.Status, result.RetryAt = "retry", next
	return result, nil
}

func pushProviderHost(endpoint string) string {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "unknown"
	}
	return strings.ToLower(parsed.Hostname())
}

func notificationPayload(delivery types.WebPushDelivery) ([]byte, error) {
	payload := map[string]string{
		"title": "Expense Tracker",
		"body":  "New activity is available",
		"tag":   "expense-tracker-activity",
	}
	if delivery.Subscription.ShowDetails {
		payload["body"] = fmt.Sprintf("New expense in %s: %s %s", delivery.GroupName, delivery.Currency, delivery.Amount)
	}
	return json.Marshal(payload)
}

func retryDelay(response *http.Response) time.Duration {
	if response != nil {
		if value := strings.TrimSpace(response.Header.Get("Retry-After")); value != "" {
			if seconds, err := time.ParseDuration(value + "s"); err == nil && seconds > 0 {
				return seconds
			}
			if retryAt, err := http.ParseTime(value); err == nil {
				return max(time.Second, time.Until(retryAt))
			}
		}
	}
	return time.Minute
}

func publicPushDialer(ctx context.Context, network, address string) (net.Conn, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil || port != "443" || !supportedPushHost(host) {
		return nil, errors.New("refusing an unsupported push destination")
	}
	lookupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	addresses, err := net.DefaultResolver.LookupNetIP(lookupCtx, "ip", host)
	if err != nil || len(addresses) == 0 {
		return nil, errors.New("refusing an unresolvable push destination")
	}
	for _, candidate := range addresses {
		if !isPublicAddress(candidate) {
			return nil, errors.New("refusing a non-public push destination")
		}
	}
	if !isPublicAddress(addresses[0]) {
		return nil, errors.New("refusing a non-public push destination")
	}
	return (&net.Dialer{Timeout: senderTimeout}).DialContext(ctx, network, net.JoinHostPort(addresses[0].String(), port))
}
