package ocr

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

const (
	AccountIDHeader  = "X-OCR-Account-ID"
	OCRRequestHeader = "X-OCR-Request-ID"
)

type StubHandler struct {
	secret         []byte
	frontendOrigin string
	replay         ReplayStore
	now            func() time.Time
}

type stubResponse struct {
	Status          string `json:"status"`
	RequestID       string `json:"requestId"`
	ProviderInvoked bool   `json:"providerInvoked"`
}

type errorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code"`
}

func NewStubHandler(config RuntimeConfig, replay ReplayStore) *StubHandler {
	return &StubHandler{
		secret:         append([]byte(nil), config.CapabilitySecret...),
		frontendOrigin: config.FrontendOrigin,
		replay:         replay,
		now:            time.Now,
	}
}

func (h *StubHandler) Handle(ctx context.Context, request events.APIGatewayV2HTTPRequest) (events.APIGatewayV2HTTPResponse, error) {
	origin := header(request.Headers, "Origin")
	if !strings.EqualFold(origin, h.frontendOrigin) {
		return h.error(http.StatusForbidden, "invalid_origin", "request origin is not allowed", ""), nil
	}
	contentType, _, err := mime.ParseMediaType(header(request.Headers, "Content-Type"))
	if err != nil || !SupportedContentType(contentType) {
		return h.error(http.StatusUnsupportedMediaType, "unsupported_media_type", "receipt must be JPEG or PNG", origin), nil
	}
	tokenString, ok := bearerToken(header(request.Headers, "Authorization"))
	if !ok {
		return h.error(http.StatusUnauthorized, "invalid_ocr_capability", "OCR capability is missing or invalid", origin), nil
	}
	claims, err := VerifyCapability(h.secret, tokenString, h.now())
	if err != nil {
		return h.error(http.StatusUnauthorized, "invalid_ocr_capability", "OCR capability is missing or invalid", origin), nil
	}
	if header(request.Headers, AccountIDHeader) != claims.Subject ||
		header(request.Headers, OCRRequestHeader) != claims.ID {
		return h.error(http.StatusForbidden, "ocr_capability_mismatch", "OCR capability does not match the request", origin), nil
	}
	if !strings.EqualFold(contentType, claims.ContentType) {
		return h.error(http.StatusForbidden, "ocr_capability_mismatch", "OCR capability does not match the request", origin), nil
	}

	document, err := requestBody(request, claims.MaxBytes)
	if err != nil {
		if errors.Is(err, errDocumentTooLarge) {
			return h.error(http.StatusRequestEntityTooLarge, "receipt_too_large", "receipt exceeds the upload limit", origin), nil
		}
		return h.error(http.StatusBadRequest, "invalid_receipt_body", "receipt body is invalid", origin), nil
	}
	if len(document) == 0 {
		return h.error(http.StatusBadRequest, "invalid_receipt_body", "receipt body is invalid", origin), nil
	}

	if err := h.replay.Claim(ctx, claims.ID, claims.ExpiresAt.Time); err != nil {
		if errors.Is(err, ErrCapabilityReplay) {
			return h.error(http.StatusConflict, "ocr_capability_replayed", "OCR capability has already been used", origin), nil
		}
		return h.error(http.StatusServiceUnavailable, "ocr_temporarily_unavailable", "receipt scanning is temporarily unavailable", origin), nil
	}

	return h.json(http.StatusOK, stubResponse{
		Status:          "stub",
		RequestID:       claims.ID,
		ProviderInvoked: false,
	}, origin), nil
}

var errDocumentTooLarge = errors.New("document too large")

func requestBody(request events.APIGatewayV2HTTPRequest, maxBytes int64) ([]byte, error) {
	if maxBytes <= 0 {
		return nil, errDocumentTooLarge
	}
	if !request.IsBase64Encoded {
		if int64(len(request.Body)) > maxBytes {
			return nil, errDocumentTooLarge
		}
		return []byte(request.Body), nil
	}
	maxEncoded := base64.StdEncoding.EncodedLen(int(maxBytes))
	if len(request.Body) > maxEncoded {
		return nil, errDocumentTooLarge
	}
	decoded, err := base64.StdEncoding.DecodeString(request.Body)
	if err != nil {
		return nil, err
	}
	if int64(len(decoded)) > maxBytes {
		return nil, errDocumentTooLarge
	}
	return decoded, nil
}

func bearerToken(value string) (string, bool) {
	scheme, token, found := strings.Cut(strings.TrimSpace(value), " ")
	return token, found && strings.EqualFold(scheme, "Bearer") && token != "" && !strings.Contains(token, " ")
}

func header(headers map[string]string, name string) string {
	for key, value := range headers {
		if strings.EqualFold(key, name) {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func (h *StubHandler) error(status int, code, message, origin string) events.APIGatewayV2HTTPResponse {
	return h.json(status, errorResponse{Error: message, Code: code}, origin)
}

func (h *StubHandler) json(status int, payload any, origin string) events.APIGatewayV2HTTPResponse {
	body, _ := json.Marshal(payload)
	headers := map[string]string{
		"Cache-Control": "no-store",
		"Content-Type":  "application/json",
	}
	if origin != "" {
		headers["Access-Control-Allow-Origin"] = origin
		headers["Vary"] = "Origin"
	}
	return events.APIGatewayV2HTTPResponse{StatusCode: status, Headers: headers, Body: string(body)}
}
