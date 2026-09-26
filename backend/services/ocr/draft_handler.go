package ocr

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"mime"
	"net/http"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
)

const (
	defaultPreprocessTimeout = 2 * time.Second
	defaultProviderTimeout   = 8 * time.Second
)

// DraftObservation contains privacy-safe operational measurements only.
type DraftObservation struct {
	Outcome                 string
	SourceBytes             int
	NormalizedBytes         int
	Width                   int
	Height                  int
	PreprocessLatencyMillis int64
	ProviderLatencyMillis   int64
	ProviderInvoked         bool
}

type DraftObserver func(DraftObservation)

type DraftHandler struct {
	secret            []byte
	frontendOrigin    string
	replay            ReplayStore
	provider          Provider
	preprocessOptions PreprocessOptions
	preprocessTimeout time.Duration
	providerTimeout   time.Duration
	observe           DraftObserver
	now               func() time.Time
}

type DraftResponse struct {
	RequestID string `json:"requestId"`
	Draft     Result `json:"draft"`
}

func NewDraftHandler(
	config RuntimeConfig,
	replay ReplayStore,
	provider Provider,
	observe DraftObserver,
) *DraftHandler {
	options := DefaultPreprocessOptions()
	options.MaxSourceBytes = MaxDocumentBytes
	return &DraftHandler{
		secret:            append([]byte(nil), config.CapabilitySecret...),
		frontendOrigin:    config.FrontendOrigin,
		replay:            replay,
		provider:          provider,
		preprocessOptions: options,
		preprocessTimeout: defaultPreprocessTimeout,
		providerTimeout:   defaultProviderTimeout,
		observe:           observe,
		now:               time.Now,
	}
}

func (h *DraftHandler) Handle(ctx context.Context, request events.APIGatewayV2HTTPRequest) (response events.APIGatewayV2HTTPResponse, err error) {
	observation := DraftObservation{Outcome: "rejected"}
	defer func() {
		if h.observe != nil {
			h.observe(observation)
		}
	}()

	origin := header(request.Headers, "Origin")
	if !strings.EqualFold(origin, h.frontendOrigin) {
		return h.error(http.StatusForbidden, "invalid_origin", "request origin is not allowed", ""), nil
	}
	contentType, _, parseErr := mime.ParseMediaType(header(request.Headers, "Content-Type"))
	if parseErr != nil || !SupportedContentType(contentType) {
		return h.error(http.StatusUnsupportedMediaType, "unsupported_media_type", "receipt must be JPEG or PNG", origin), nil
	}
	tokenString, ok := bearerToken(header(request.Headers, "Authorization"))
	if !ok {
		return h.error(http.StatusUnauthorized, "invalid_ocr_capability", "OCR capability is missing or invalid", origin), nil
	}
	claims, verifyErr := VerifyCapability(h.secret, tokenString, h.now())
	if verifyErr != nil {
		return h.error(http.StatusUnauthorized, "invalid_ocr_capability", "OCR capability is missing or invalid", origin), nil
	}
	if header(request.Headers, AccountIDHeader) != claims.Subject ||
		header(request.Headers, OCRRequestHeader) != claims.ID ||
		!strings.EqualFold(contentType, claims.ContentType) {
		return h.error(http.StatusForbidden, "ocr_capability_mismatch", "OCR capability does not match the request", origin), nil
	}

	document, bodyErr := requestBody(request, claims.MaxBytes)
	if bodyErr != nil {
		if errors.Is(bodyErr, errDocumentTooLarge) {
			return h.error(http.StatusRequestEntityTooLarge, "receipt_too_large", "receipt exceeds the upload limit", origin), nil
		}
		return h.error(http.StatusBadRequest, "invalid_receipt_body", "receipt body is invalid", origin), nil
	}
	if len(document) == 0 {
		return h.error(http.StatusBadRequest, "invalid_receipt_body", "receipt body is invalid", origin), nil
	}
	observation.SourceBytes = len(document)

	preprocessStarted := time.Now()
	preprocessContext, cancelPreprocess := context.WithTimeout(ctx, h.preprocessTimeout)
	processed, preprocessErr := PreprocessRasterContext(
		preprocessContext,
		bytes.NewReader(document),
		h.preprocessOptions,
	)
	cancelPreprocess()
	observation.PreprocessLatencyMillis = time.Since(preprocessStarted).Milliseconds()
	if preprocessErr != nil {
		observation.Outcome = "invalid_image"
		return h.preprocessError(preprocessErr, origin), nil
	}
	if !strings.EqualFold(contentType, mediaTypeForFormat(processed.SourceFormat)) {
		observation.Outcome = "media_type_mismatch"
		return h.error(http.StatusUnsupportedMediaType, "receipt_type_mismatch", "receipt content does not match its media type", origin), nil
	}
	observation.NormalizedBytes = len(processed.Bytes)
	observation.Width = processed.Width
	observation.Height = processed.Height

	if claimErr := h.replay.Claim(ctx, claims.ID, claims.ExpiresAt.Time); claimErr != nil {
		if errors.Is(claimErr, ErrCapabilityReplay) {
			observation.Outcome = "replay"
			return h.error(http.StatusConflict, "ocr_capability_replayed", "OCR capability has already been used", origin), nil
		}
		observation.Outcome = "replay_store_error"
		return h.retryableError("receipt scanning is temporarily unavailable", origin), nil
	}

	providerStarted := time.Now()
	providerContext, cancelProvider := context.WithTimeout(ctx, h.providerTimeout)
	observation.ProviderInvoked = true
	extraction, providerErr := h.provider.AnalyzeExpense(providerContext, processed.Bytes)
	cancelProvider()
	observation.ProviderLatencyMillis = time.Since(providerStarted).Milliseconds()
	if providerErr != nil {
		if errors.Is(providerErr, context.DeadlineExceeded) || errors.Is(providerContext.Err(), context.DeadlineExceeded) {
			observation.Outcome = "provider_timeout"
			return h.retryableError("receipt scanning timed out; you can retry or enter the expense manually", origin), nil
		}
		if errors.Is(providerErr, ErrProviderRejected) {
			observation.Outcome = "provider_rejected"
			return h.error(http.StatusUnprocessableEntity, "receipt_not_recognized", "the receipt could not be recognized; try another image or enter the expense manually", origin), nil
		}
		observation.Outcome = "provider_error"
		return h.retryableError("receipt scanning is temporarily unavailable; you can retry or enter the expense manually", origin), nil
	}

	observation.Outcome = "success"
	return h.json(http.StatusOK, DraftResponse{
		RequestID: claims.ID,
		Draft:     PrepareDraft(extraction),
	}, origin), nil
}

func (h *DraftHandler) preprocessError(err error, origin string) events.APIGatewayV2HTTPResponse {
	switch {
	case errors.Is(err, ErrInputTooLarge), errors.Is(err, ErrOutputTooLarge):
		return h.error(http.StatusRequestEntityTooLarge, "receipt_too_large", "receipt exceeds the processing limit", origin)
	case errors.Is(err, ErrDimensionsTooLarge):
		return h.error(http.StatusUnprocessableEntity, "receipt_dimensions_exceeded", "receipt dimensions exceed the processing limit", origin)
	case errors.Is(err, ErrPreprocessTimeout):
		return h.error(http.StatusRequestTimeout, "receipt_processing_timeout", "receipt processing timed out", origin)
	default:
		return h.error(http.StatusUnprocessableEntity, "invalid_receipt_image", "receipt image could not be processed", origin)
	}
}

func mediaTypeForFormat(format string) string {
	switch format {
	case "jpeg":
		return "image/jpeg"
	case "png":
		return "image/png"
	default:
		return ""
	}
}

func (h *DraftHandler) retryableError(message, origin string) events.APIGatewayV2HTTPResponse {
	response := h.error(http.StatusServiceUnavailable, "ocr_temporarily_unavailable", message, origin)
	response.Headers["Retry-After"] = "1"
	return response
}

func (h *DraftHandler) error(status int, code, message, origin string) events.APIGatewayV2HTTPResponse {
	return h.json(status, errorResponse{Error: message, Code: code}, origin)
}

func (h *DraftHandler) json(status int, payload any, origin string) events.APIGatewayV2HTTPResponse {
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
