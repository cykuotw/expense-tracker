package errornotifier

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWebhookProviderValidatesOnceAndReturnsCopies(t *testing.T) {
	provider, err := NewWebhookProvider(testWebhook)
	require.NoError(t, err)

	first, err := provider.URL(t.Context())
	require.NoError(t, err)
	first.Path = "/changed"
	second, err := provider.URL(t.Context())
	require.NoError(t, err)
	require.Equal(t, testWebhook, second.String())
}

func TestWebhookProviderRejectsInvalidValueWithoutExposingIt(t *testing.T) {
	value := "https://example.com/CANARY_WEBHOOK_SECRET"
	_, err := NewWebhookProvider(value)
	require.Error(t, err)
	require.False(t, IsRetryable(err))
	require.NotContains(t, err.Error(), "CANARY_WEBHOOK_SECRET")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return f(request)
}

type trackingBody struct {
	reader io.Reader
	read   int
	closed bool
}

func (b *trackingBody) Read(value []byte) (int, error) {
	read, err := b.reader.Read(value)
	b.read += read
	return read, err
}

func (b *trackingBody) Close() error {
	b.closed = true
	return nil
}

func TestDiscordClientDisablesMentionsBoundsAndClosesResponse(t *testing.T) {
	responseBody := &trackingBody{reader: strings.NewReader("ok")}
	var requestBody []byte
	client, err := NewDiscordClient(&http.Client{Transport: roundTripFunc(
		func(request *http.Request) (*http.Response, error) {
			var readErr error
			requestBody, readErr = io.ReadAll(request.Body)
			require.NoError(t, readErr)
			require.Equal(t, "application/json", request.Header.Get("Content-Type"))
			return &http.Response{StatusCode: http.StatusNoContent, Body: responseBody}, nil
		},
	)})
	require.NoError(t, err)
	webhook, err := url.Parse(testWebhook)
	require.NoError(t, err)
	require.NoError(t, client.Post(t.Context(), webhook, "safe message"))
	require.True(t, responseBody.closed)

	var payload struct {
		Content         string `json:"content"`
		AllowedMentions struct {
			Parse []string `json:"parse"`
		} `json:"allowed_mentions"`
	}
	require.NoError(t, json.Unmarshal(requestBody, &payload))
	require.Equal(t, "safe message", payload.Content)
	require.NotNil(t, payload.AllowedMentions.Parse)
	require.Empty(t, payload.AllowedMentions.Parse)
}

func TestDiscordClientClassifiesStatusTransportAndResponseBounds(t *testing.T) {
	webhook, err := url.Parse(testWebhook)
	require.NoError(t, err)
	for name, test := range map[string]struct {
		status       int
		transportErr error
		oversized    bool
		retry        bool
	}{
		"bad request":  {status: http.StatusBadRequest},
		"rate limited": {status: http.StatusTooManyRequests, retry: true},
		"unavailable":  {status: http.StatusServiceUnavailable, retry: true},
		"transport": {
			transportErr: errors.New("CANARY_WEBHOOK_SECRET"),
			retry:        true,
		},
		"oversized": {status: http.StatusNoContent, oversized: true},
	} {
		t.Run(name, func(t *testing.T) {
			body := &trackingBody{reader: strings.NewReader("response")}
			if test.oversized {
				body.reader = strings.NewReader(strings.Repeat("x", maxResponseBodyBytes+100))
			}
			client, createErr := NewDiscordClient(&http.Client{Transport: roundTripFunc(
				func(*http.Request) (*http.Response, error) {
					if test.transportErr != nil {
						return nil, test.transportErr
					}
					return &http.Response{StatusCode: test.status, Body: body}, nil
				},
			)})
			require.NoError(t, createErr)
			err := client.Post(t.Context(), webhook, "safe")
			require.Error(t, err)
			require.Equal(t, test.retry, IsRetryable(err))
			require.NotContains(t, err.Error(), "CANARY_WEBHOOK_SECRET")
			if test.transportErr == nil {
				require.True(t, body.closed)
				require.LessOrEqual(t, body.read, maxResponseBodyBytes+1)
			}
		})
	}
}

func TestDiscordHTTPClientHasOverallTimeoutAndRejectsRedirects(t *testing.T) {
	client := NewHTTPClient()
	require.Equal(t, discordHTTPTimeout, client.Timeout)
	redirect, err := http.NewRequest(http.MethodGet, "https://example.com/redirect", nil)
	require.NoError(t, err)
	require.ErrorIs(t, client.CheckRedirect(redirect, nil), http.ErrUseLastResponse)

	client.Timeout = 10 * time.Millisecond
	client.Transport = roundTripFunc(func(request *http.Request) (*http.Response, error) {
		<-request.Context().Done()
		return nil, request.Context().Err()
	})
	discord, err := NewDiscordClient(client)
	require.NoError(t, err)
	webhook, err := url.Parse(testWebhook)
	require.NoError(t, err)
	err = discord.Post(context.Background(), webhook, "safe")
	require.Error(t, err)
	require.True(t, IsRetryable(err))
}

func TestConfigurationValidationRejectsUnsafeDestinationsAndLabels(t *testing.T) {
	for _, value := range []string{
		"http://discord.com/api/webhooks/123/abcdefghijklmnopqrstuvwxyz",
		"https://example.com/api/webhooks/123/abcdefghijklmnopqrstuvwxyz",
		"https://discord.com/api/other/123/abcdefghijklmnopqrstuvwxyz",
		"https://discord.com/api/webhooks/123/short",
	} {
		_, err := parseDiscordWebhookURL(value)
		require.Error(t, err)
		require.NotContains(t, err.Error(), value)
	}
	_, err := NewHandler(
		"production @everyone",
		"function",
		&fakeWebhookSource{},
		&fakeDiscordPoster{},
	)
	require.Error(t, err)
}

func TestDiscordClientRejectsInvalidMessageBoundsBeforeTransport(t *testing.T) {
	client, err := NewDiscordClient(&http.Client{Transport: roundTripFunc(
		func(*http.Request) (*http.Response, error) {
			t.Fatal("transport must not be called")
			return nil, nil
		},
	)})
	require.NoError(t, err)
	webhook, err := url.Parse(testWebhook)
	require.NoError(t, err)
	for _, message := range []string{"", strings.Repeat("x", maxDiscordMessageBytes+1)} {
		err := client.Post(t.Context(), webhook, message)
		require.Error(t, err)
		require.False(t, IsRetryable(err))
	}
}
