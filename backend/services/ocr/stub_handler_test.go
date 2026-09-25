package ocr

import (
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/google/uuid"
)

type replayStub struct {
	claimed string
	err     error
}

func (s *replayStub) Claim(_ context.Context, requestID string, _ time.Time) error {
	s.claimed = requestID
	return s.err
}

func stubRequest(token, accountID, requestID, contentType string, body []byte) events.APIGatewayV2HTTPRequest {
	return events.APIGatewayV2HTTPRequest{
		Headers: map[string]string{
			"Origin":         "https://app.example.com",
			"Authorization":  "Bearer " + token,
			"Content-Type":   contentType,
			AccountIDHeader:  accountID,
			OCRRequestHeader: requestID,
		},
		Body:            base64.StdEncoding.EncodeToString(body),
		IsBase64Encoded: true,
	}
}

func TestStubHandlerAcceptsOneBoundRequestWithoutProviderCall(t *testing.T) {
	now := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	secret := []byte(strings.Repeat("s", 32))
	accountID := uuid.NewString()
	requestID := uuid.NewString()
	token, _, err := MintCapability(secret, now, accountID, requestID, "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	replay := &replayStub{}
	handler := NewStubHandler(RuntimeConfig{
		FrontendOrigin: "https://app.example.com", CapabilitySecret: secret,
	}, replay)
	handler.now = func() time.Time { return now.Add(time.Second) }

	response, err := handler.Handle(t.Context(), stubRequest(token, accountID, requestID, "image/jpeg", []byte("phase-two-stub")))
	if err != nil {
		t.Fatal(err)
	}
	if response.StatusCode != http.StatusOK || replay.claimed != requestID ||
		!strings.Contains(response.Body, `"providerInvoked":false`) ||
		response.Headers["Cache-Control"] != "no-store" {
		t.Fatalf("unexpected response: %#v", response)
	}
}

func TestStubHandlerRejectsInvalidBoundaryBeforeReplayClaim(t *testing.T) {
	now := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	secret := []byte(strings.Repeat("s", 32))
	accountID := uuid.NewString()
	requestID := uuid.NewString()
	token, _, err := MintCapability(secret, now, accountID, requestID, "image/png")
	if err != nil {
		t.Fatal(err)
	}

	tests := map[string]struct {
		request events.APIGatewayV2HTTPRequest
		status  int
	}{
		"wrong origin": {
			request: stubRequest(token, accountID, requestID, "image/png", []byte("stub")),
			status:  http.StatusForbidden,
		},
		"wrong account": {
			request: stubRequest(token, uuid.NewString(), requestID, "image/png", []byte("stub")),
			status:  http.StatusForbidden,
		},
		"wrong request": {
			request: stubRequest(token, accountID, uuid.NewString(), "image/png", []byte("stub")),
			status:  http.StatusForbidden,
		},
		"wrong type": {
			request: stubRequest(token, accountID, requestID, "image/jpeg", []byte("stub")),
			status:  http.StatusForbidden,
		},
		"oversized": {
			request: stubRequest(token, accountID, requestID, "image/png", make([]byte, MaxDocumentBytes+1)),
			status:  http.StatusRequestEntityTooLarge,
		},
	}
	wrongOrigin := tests["wrong origin"]
	wrongOrigin.request.Headers["Origin"] = "https://attacker.example.com"
	tests["wrong origin"] = wrongOrigin

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			replay := &replayStub{}
			handler := NewStubHandler(RuntimeConfig{
				FrontendOrigin: "https://app.example.com", CapabilitySecret: secret,
			}, replay)
			handler.now = func() time.Time { return now.Add(time.Second) }
			response, err := handler.Handle(t.Context(), test.request)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != test.status || replay.claimed != "" {
				t.Fatalf("response = %#v, claimed = %q", response, replay.claimed)
			}
		})
	}
}

func TestStubHandlerRejectsReplayAndStoreFailureSafely(t *testing.T) {
	now := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	secret := []byte(strings.Repeat("s", 32))
	accountID := uuid.NewString()
	requestID := uuid.NewString()
	token, _, err := MintCapability(secret, now, accountID, requestID, "image/jpeg")
	if err != nil {
		t.Fatal(err)
	}
	request := stubRequest(token, accountID, requestID, "image/jpeg", []byte("stub"))
	for name, test := range map[string]struct {
		err    error
		status int
	}{
		"replay":  {ErrCapabilityReplay, http.StatusConflict},
		"failure": {errors.New("sensitive backend detail"), http.StatusServiceUnavailable},
	} {
		t.Run(name, func(t *testing.T) {
			handler := NewStubHandler(RuntimeConfig{
				FrontendOrigin: "https://app.example.com", CapabilitySecret: secret,
			}, &replayStub{err: test.err})
			handler.now = func() time.Time { return now.Add(time.Second) }
			response, err := handler.Handle(t.Context(), request)
			if err != nil {
				t.Fatal(err)
			}
			if response.StatusCode != test.status || strings.Contains(response.Body, "sensitive backend detail") {
				t.Fatalf("unexpected response: %#v", response)
			}
		})
	}
}
