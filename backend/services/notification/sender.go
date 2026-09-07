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
	"strings"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
)

const (
	senderBatchSize = 10
	senderTimeout   = 5 * time.Second
	maxAttempts     = 3
)

type Sender struct {
	store      *Store
	publicKey  string
	privateKey string
	subscriber string
	client     *http.Client
	clock      func() time.Time
}

func NewSender(store *Store, publicKey, privateKey, subscriber string) (*Sender, error) {
	if strings.TrimSpace(publicKey) == "" || strings.TrimSpace(privateKey) == "" || !strings.HasPrefix(subscriber, "mailto:") {
		return nil, errors.New("WEB_PUSH_VAPID_PUBLIC_KEY, WEB_PUSH_VAPID_PRIVATE_KEY, and WEB_PUSH_VAPID_SUBJECT are required")
	}
	return &Sender{
		store:      store,
		publicKey:  publicKey,
		privateKey: privateKey,
		subscriber: subscriber,
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
	if err := s.store.Cleanup(ctx); err != nil {
		return fmt.Errorf("cleanup: %w", err)
	}
	deliveries, err := s.store.GetPendingDeliveries(ctx, senderBatchSize)
	if err != nil {
		return fmt.Errorf("load pending deliveries: %w", err)
	}
	for _, delivery := range deliveries {
		if err := s.send(ctx, delivery); err != nil {
			return err
		}
	}
	return nil
}

func (s *Sender) send(ctx context.Context, delivery types.WebPushDelivery) error {
	payload, err := notificationPayload(delivery)
	if err != nil {
		return fmt.Errorf("marshal notification payload: %w", err)
	}
	ttl := max(1, int(time.Until(delivery.ExpiresAt).Seconds()))
	response, sendErr := webpush.SendNotification(payload, &webpush.Subscription{
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
		return s.store.CompleteDelivery(ctx, delivery.ID, "delivered")
	}
	if response != nil && (response.StatusCode == http.StatusNotFound || response.StatusCode == http.StatusGone) {
		return s.store.RetireSubscription(ctx, delivery.Subscription.ID)
	}
	if response != nil && response.StatusCode >= http.StatusBadRequest && response.StatusCode < http.StatusInternalServerError && response.StatusCode != http.StatusTooManyRequests {
		return s.store.CompleteDelivery(ctx, delivery.ID, "provider_rejected")
	}

	attempts := delivery.Attempts + 1
	next := s.clock().UTC().Add(retryDelay(response))
	if attempts >= maxAttempts || !next.Before(delivery.ExpiresAt) {
		return s.store.CompleteDelivery(ctx, delivery.ID, "failed")
	}
	if err := s.store.RetryDelivery(ctx, delivery.ID, next, attempts); err != nil {
		return fmt.Errorf("schedule delivery retry: %w", err)
	}
	if sendErr != nil {
		slog.Warn("web push delivery deferred", "attempt", attempts)
	}
	return nil
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
