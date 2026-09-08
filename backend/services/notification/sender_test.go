package notification

import (
	"context"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"errors"
	"expense-tracker/backend/types"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	webpush "github.com/SherClockHolmes/webpush-go"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type fakeDeliveryStore struct {
	batch    DeliveryBatch
	claimErr error
	ackErr   error
	acks     int
	results  []DeliveryResult
}

func (s *fakeDeliveryStore) ClaimDeliveries(context.Context) (DeliveryBatch, error) {
	return s.batch, s.claimErr
}
func (s *fakeDeliveryStore) AcknowledgeDeliveries(_ context.Context, token uuid.UUID, results []DeliveryResult) error {
	if token != s.batch.Token {
		return errors.New("wrong claim token")
	}
	s.acks++
	s.results = results
	return s.ackErr
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func testDelivery(t *testing.T) types.WebPushDelivery {
	t.Helper()
	key, err := ecdh.P256().GenerateKey(rand.Reader)
	require.NoError(t, err)
	return types.WebPushDelivery{
		ID: uuid.New(), ExpiresAt: time.Now().Add(time.Hour),
		Subscription: types.WebPushSubscription{
			ID: uuid.New(), Endpoint: "https://fcm.googleapis.com/push/test",
			P256DH: base64.RawURLEncoding.EncodeToString(key.PublicKey().Bytes()),
			Auth:   base64.RawURLEncoding.EncodeToString(make([]byte, 16)),
		},
	}
}

func testSender(t *testing.T, store DeliveryStore, transport roundTripFunc) *Sender {
	t.Helper()
	private, public, err := webpush.GenerateVAPIDKeys()
	require.NoError(t, err)
	sender, err := NewSender(store, public, private, "mailto:ops@example.com")
	require.NoError(t, err)
	sender.client.Transport = transport
	return sender
}

func TestSenderAcknowledgesBatchOutcomes(t *testing.T) {
	for _, tc := range []struct {
		name   string
		code   int
		status string
	}{
		{"success", 201, "delivered"}, {"gone", 410, "retired"}, {"missing", 404, "retired"},
		{"invalid", 400, "provider_rejected"}, {"limited", 429, "retry"}, {"unavailable", 503, "retry"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			store := &fakeDeliveryStore{batch: DeliveryBatch{Token: uuid.New(), Deliveries: []types.WebPushDelivery{testDelivery(t), testDelivery(t)}}}
			sender := testSender(t, store, func(r *http.Request) (*http.Response, error) {
				require.Equal(t, http.MethodPost, r.Method)
				return &http.Response{StatusCode: tc.code, Header: http.Header{"Retry-After": {"120"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
			})
			require.NoError(t, sender.RunOnce(t.Context()))
			require.Equal(t, 1, store.acks)
			require.Len(t, store.results, 2)
			for i, result := range store.results {
				require.Equal(t, store.batch.Deliveries[i].ID, result.ID)
				require.Equal(t, tc.status, result.Status)
				if tc.status == "retry" {
					require.WithinDuration(t, time.Now().Add(120*time.Second), result.RetryAt, time.Second)
				}
			}
		})
	}
}

func TestSenderEmptyAndClaimFailureDoNotAcknowledge(t *testing.T) {
	for _, claimErr := range []error{nil, errors.New("database unavailable")} {
		store := &fakeDeliveryStore{batch: DeliveryBatch{Token: uuid.New()}, claimErr: claimErr}
		sender := testSender(t, store, func(*http.Request) (*http.Response, error) { t.Fatal("unexpected push"); return nil, nil })
		err := sender.RunOnce(t.Context())
		require.ErrorIs(t, err, claimErr)
		require.Zero(t, store.acks)
	}
}

func TestSenderRetryLimitAndAcknowledgementFailure(t *testing.T) {
	delivery := testDelivery(t)
	delivery.Attempts = maxAttempts - 1
	ackErr := errors.New("ack unavailable")
	store := &fakeDeliveryStore{batch: DeliveryBatch{Token: uuid.New(), Deliveries: []types.WebPushDelivery{delivery}}, ackErr: ackErr}
	sender := testSender(t, store, func(*http.Request) (*http.Response, error) { return nil, errors.New("network unavailable") })
	require.ErrorIs(t, sender.RunOnce(t.Context()), ackErr)
	require.Equal(t, "failed", store.results[0].Status)
}
