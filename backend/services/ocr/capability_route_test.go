package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type grantStoreStub struct {
	granted bool
	err     error
	userID  string
}

func (s *grantStoreStub) HasReceiptOCRGrant(_ context.Context, userID string) (bool, error) {
	s.userID = userID
	return s.granted, s.err
}

func capabilityRouteRequest(t *testing.T, userID uuid.UUID, body string) *http.Request {
	t.Helper()
	token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), userID)
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/ocr/capabilities", bytes.NewBufferString(body))
	request.Header.Set("Content-Type", "application/json")
	request.AddCookie(&http.Cookie{Name: "access_token", Value: token})
	return request
}

func capabilityRoute(handler *CapabilityHandler, request *http.Request) *httptest.ResponseRecorder {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	handler.RegisterRoutes(router.Group(""))
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response
}

func TestCapabilityRouteMintsAccountBoundToken(t *testing.T) {
	now := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	userID := uuid.New()
	requestID := uuid.NewString()
	store := &grantStoreStub{granted: true}
	handler := NewCapabilityHandler(store, []byte(strings.Repeat("o", 32)))
	handler.now = func() time.Time { return now }

	response := capabilityRoute(handler, capabilityRouteRequest(
		t,
		userID,
		`{"requestId":"`+requestID+`","contentType":"image/jpeg"}`,
	))
	if response.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
	}
	if store.userID != userID.String() {
		t.Fatalf("grant checked for %q", store.userID)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		t.Fatal("missing no-store response header")
	}
	var payload CapabilityResponse
	if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	claims, err := VerifyCapability([]byte(strings.Repeat("o", 32)), payload.Token, now.Add(time.Second))
	if err != nil {
		t.Fatal(err)
	}
	if claims.Subject != userID.String() || claims.ID != requestID || payload.MaxBytes != MaxDocumentBytes {
		t.Fatalf("unexpected capability response: %#v, %#v", payload, claims)
	}
}

func TestCapabilityRouteRejectsMissingGrantAndInvalidRequests(t *testing.T) {
	userID := uuid.New()
	validBody := `{"requestId":"` + uuid.NewString() + `","contentType":"image/png"}`

	t.Run("missing grant", func(t *testing.T) {
		response := capabilityRoute(
			NewCapabilityHandler(&grantStoreStub{}, []byte(strings.Repeat("o", 32))),
			capabilityRouteRequest(t, userID, validBody),
		)
		if response.Code != http.StatusForbidden || !strings.Contains(response.Body.String(), "receipt_ocr_not_granted") {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
	})

	t.Run("store failure is safe", func(t *testing.T) {
		response := capabilityRoute(
			NewCapabilityHandler(&grantStoreStub{err: errors.New("database detail")}, []byte(strings.Repeat("o", 32))),
			capabilityRouteRequest(t, userID, validBody),
		)
		if response.Code != http.StatusInternalServerError || strings.Contains(response.Body.String(), "database detail") {
			t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
		}
	})

	for name, body := range map[string]string{
		"unknown field":  `{"requestId":"` + uuid.NewString() + `","contentType":"image/png","accountId":"attacker"}`,
		"bad request ID": `{"requestId":"not-a-uuid","contentType":"image/png"}`,
		"bad type":       `{"requestId":"` + uuid.NewString() + `","contentType":"image/webp"}`,
	} {
		t.Run(name, func(t *testing.T) {
			response := capabilityRoute(
				NewCapabilityHandler(&grantStoreStub{granted: true}, []byte(strings.Repeat("o", 32))),
				capabilityRouteRequest(t, userID, body),
			)
			if response.Code != http.StatusBadRequest {
				t.Fatalf("status = %d, body = %s", response.Code, response.Body.String())
			}
		})
	}
}
