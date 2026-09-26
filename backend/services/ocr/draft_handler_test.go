package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"image"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type providerStub struct {
	result   Result
	err      error
	document []byte
	calls    int
	wait     bool
}

func (stub *providerStub) AnalyzeExpense(ctx context.Context, document []byte) (Result, error) {
	stub.calls++
	stub.document = append([]byte(nil), document...)
	if stub.wait {
		<-ctx.Done()
		return Result{}, ctx.Err()
	}
	return stub.result, stub.err
}

func draftFixture(t *testing.T, contentType string) (time.Time, string, string, string, []byte) {
	t.Helper()
	now := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	secret := []byte(strings.Repeat("s", 32))
	accountID := uuid.NewString()
	requestID := uuid.NewString()
	token, _, err := MintCapability(secret, now, accountID, requestID, contentType)
	require.NoError(t, err)
	return now, accountID, requestID, token, secret
}

func TestDraftHandlerReturnsNormalizedEditableDraft(t *testing.T) {
	now, accountID, requestID, token, secret := draftFixture(t, "image/jpeg")
	provider := &providerStub{result: Result{
		Merchant: Field{Value: "Example Market", Confidence: 91},
		Subtotal: Field{Value: "$4.00", Confidence: 95},
		Total:    Field{Value: "$4.00", Confidence: 96},
		Items: []Item{{
			Description: Field{Value: "Milk", Confidence: 98},
			UnitPrice:   Field{Value: "$4.00", Confidence: 90},
			LineTotal:   Field{Value: "$4.00", Confidence: 97},
		}},
	}}
	replay := &replayStub{}
	var observation DraftObservation
	handler := NewDraftHandler(RuntimeConfig{
		FrontendOrigin: "https://app.example.com", CapabilitySecret: secret,
	}, replay, provider, func(value DraftObservation) { observation = value })
	handler.now = func() time.Time { return now.Add(time.Second) }

	request := stubRequest(token, accountID, requestID, "image/jpeg", encodeTestJPEG(t, 16, 12))
	response, err := handler.Handle(t.Context(), request)
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, response.StatusCode)
	assert.Equal(t, requestID, replay.claimed)
	assert.Equal(t, 1, provider.calls)
	assert.Equal(t, "success", observation.Outcome)
	assert.True(t, observation.ProviderInvoked)
	assert.Positive(t, observation.NormalizedBytes)
	_, format, err := image.Decode(bytes.NewReader(provider.document))
	require.NoError(t, err)
	assert.Equal(t, "jpeg", format)

	var payload DraftResponse
	require.NoError(t, json.Unmarshal([]byte(response.Body), &payload))
	assert.Equal(t, requestID, payload.RequestID)
	assert.Equal(t, ProvenanceProvider, payload.Draft.Merchant.Provenance)
	assert.True(t, payload.Draft.Merchant.RequiresReview)
	assert.Equal(t, ProvenanceInferred, payload.Draft.Tax.Provenance)
	assert.Equal(t, "1", payload.Draft.Items[0].Quantity.Value)
	assert.NotContains(t, response.Body, "RawRow")
}

func TestDraftHandlerRejectsInvalidImageBeforeReplayAndProvider(t *testing.T) {
	now, accountID, requestID, token, secret := draftFixture(t, "image/jpeg")
	replay := &replayStub{}
	provider := &providerStub{}
	handler := NewDraftHandler(RuntimeConfig{
		FrontendOrigin: "https://app.example.com", CapabilitySecret: secret,
	}, replay, provider, nil)
	handler.now = func() time.Time { return now.Add(time.Second) }

	response, err := handler.Handle(t.Context(), stubRequest(token, accountID, requestID, "image/jpeg", []byte("not an image")))
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	assert.Empty(t, replay.claimed)
	assert.Zero(t, provider.calls)
}

func TestDraftHandlerRejectsInvalidBoundaryBeforeReplayAndProvider(t *testing.T) {
	now, accountID, requestID, token, secret := draftFixture(t, "image/jpeg")
	validImage := encodeTestJPEG(t, 8, 8)

	tests := map[string]struct {
		request func() events.APIGatewayV2HTTPRequest
		status  int
	}{
		"wrong origin": {
			request: func() events.APIGatewayV2HTTPRequest {
				request := stubRequest(token, accountID, requestID, "image/jpeg", validImage)
				request.Headers["Origin"] = "https://attacker.example.com"
				return request
			},
			status: http.StatusForbidden,
		},
		"wrong account": {
			request: func() events.APIGatewayV2HTTPRequest {
				return stubRequest(token, uuid.NewString(), requestID, "image/jpeg", validImage)
			},
			status: http.StatusForbidden,
		},
		"wrong request": {
			request: func() events.APIGatewayV2HTTPRequest {
				return stubRequest(token, accountID, uuid.NewString(), "image/jpeg", validImage)
			},
			status: http.StatusForbidden,
		},
		"wrong claimed type": {
			request: func() events.APIGatewayV2HTTPRequest {
				return stubRequest(token, accountID, requestID, "image/png", validImage)
			},
			status: http.StatusForbidden,
		},
		"oversized": {
			request: func() events.APIGatewayV2HTTPRequest {
				return stubRequest(token, accountID, requestID, "image/jpeg", make([]byte, MaxDocumentBytes+1))
			},
			status: http.StatusRequestEntityTooLarge,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			replay := &replayStub{}
			provider := &providerStub{}
			handler := NewDraftHandler(RuntimeConfig{
				FrontendOrigin: "https://app.example.com", CapabilitySecret: secret,
			}, replay, provider, nil)
			handler.now = func() time.Time { return now.Add(time.Second) }

			response, err := handler.Handle(t.Context(), test.request())
			require.NoError(t, err)
			assert.Equal(t, test.status, response.StatusCode)
			assert.Empty(t, replay.claimed)
			assert.Zero(t, provider.calls)
		})
	}
}

func TestDraftHandlerRejectsMislabeledImageBeforeReplayAndProvider(t *testing.T) {
	now, accountID, requestID, token, secret := draftFixture(t, "image/png")
	replay := &replayStub{}
	provider := &providerStub{}
	handler := NewDraftHandler(RuntimeConfig{
		FrontendOrigin: "https://app.example.com", CapabilitySecret: secret,
	}, replay, provider, nil)
	handler.now = func() time.Time { return now.Add(time.Second) }

	response, err := handler.Handle(t.Context(), stubRequest(token, accountID, requestID, "image/png", encodeTestJPEG(t, 8, 8)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnsupportedMediaType, response.StatusCode)
	assert.Empty(t, replay.claimed)
	assert.Zero(t, provider.calls)
}

func TestDraftHandlerPreventsDuplicateProviderCall(t *testing.T) {
	now, accountID, requestID, token, secret := draftFixture(t, "image/jpeg")
	provider := &providerStub{}
	handler := NewDraftHandler(RuntimeConfig{
		FrontendOrigin: "https://app.example.com", CapabilitySecret: secret,
	}, &replayStub{err: ErrCapabilityReplay}, provider, nil)
	handler.now = func() time.Time { return now.Add(time.Second) }

	response, err := handler.Handle(t.Context(), stubRequest(token, accountID, requestID, "image/jpeg", encodeTestJPEG(t, 8, 8)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, response.StatusCode)
	assert.Zero(t, provider.calls)
}

func TestDraftHandlerReturnsSafeRetryableProviderError(t *testing.T) {
	now, accountID, requestID, token, secret := draftFixture(t, "image/jpeg")
	provider := &providerStub{err: errors.New("sensitive provider payload")}
	handler := NewDraftHandler(RuntimeConfig{
		FrontendOrigin: "https://app.example.com", CapabilitySecret: secret,
	}, &replayStub{}, provider, nil)
	handler.now = func() time.Time { return now.Add(time.Second) }

	response, err := handler.Handle(t.Context(), stubRequest(token, accountID, requestID, "image/jpeg", encodeTestJPEG(t, 8, 8)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	assert.Equal(t, "1", response.Headers["Retry-After"])
	assert.NotContains(t, response.Body, "sensitive provider payload")
}

func TestDraftHandlerReturnsNonRetryableRejectedDocumentError(t *testing.T) {
	now, accountID, requestID, token, secret := draftFixture(t, "image/jpeg")
	provider := &providerStub{err: ErrProviderRejected}
	handler := NewDraftHandler(RuntimeConfig{
		FrontendOrigin: "https://app.example.com", CapabilitySecret: secret,
	}, &replayStub{}, provider, nil)
	handler.now = func() time.Time { return now.Add(time.Second) }

	response, err := handler.Handle(t.Context(), stubRequest(token, accountID, requestID, "image/jpeg", encodeTestJPEG(t, 8, 8)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnprocessableEntity, response.StatusCode)
	assert.Empty(t, response.Headers["Retry-After"])
	assert.Contains(t, response.Body, "receipt_not_recognized")
}

func TestDraftHandlerBoundsProviderTime(t *testing.T) {
	now, accountID, requestID, token, secret := draftFixture(t, "image/jpeg")
	provider := &providerStub{wait: true}
	handler := NewDraftHandler(RuntimeConfig{
		FrontendOrigin: "https://app.example.com", CapabilitySecret: secret,
	}, &replayStub{}, provider, nil)
	handler.now = func() time.Time { return now.Add(time.Second) }
	handler.providerTimeout = time.Millisecond

	response, err := handler.Handle(t.Context(), stubRequest(token, accountID, requestID, "image/jpeg", encodeTestJPEG(t, 8, 8)))
	require.NoError(t, err)
	assert.Equal(t, http.StatusServiceUnavailable, response.StatusCode)
	assert.Contains(t, response.Body, "timed out")
}
