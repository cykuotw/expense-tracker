package ocr

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strings"
	"time"

	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const maxCapabilityRequestBytes = 256

var (
	errInvalidCapabilityRequest = utils.NewDetailedError(
		"invalid OCR capability request",
		"invalid_ocr_capability_request",
		nil,
	)
	errReceiptOCRNotGranted = utils.NewDetailedError(
		"receipt OCR is not available for this account",
		"receipt_ocr_not_granted",
		nil,
	)
)

type ReceiptOCRGrantStore interface {
	HasReceiptOCRGrant(ctx context.Context, userID string) (bool, error)
}

type CapabilityHandler struct {
	store  ReceiptOCRGrantStore
	secret []byte
	now    func() time.Time
}

type capabilityRequest struct {
	RequestID   string `json:"requestId"`
	ContentType string `json:"contentType"`
}

type CapabilityResponse struct {
	Token       string    `json:"token"`
	RequestID   string    `json:"requestId"`
	ContentType string    `json:"contentType"`
	MaxBytes    int64     `json:"maxBytes"`
	ExpiresAt   time.Time `json:"expiresAt"`
}

func NewCapabilityHandler(store ReceiptOCRGrantStore, secret []byte) *CapabilityHandler {
	return &CapabilityHandler{
		store:  store,
		secret: append([]byte(nil), secret...),
		now:    time.Now,
	}
}

func (h *CapabilityHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/ocr/capabilities", h.handleCreate)
}

func (h *CapabilityHandler) handleCreate(c *gin.Context) {
	contentType, _, err := mime.ParseMediaType(c.GetHeader("Content-Type"))
	if err != nil || contentType != "application/json" {
		utils.WriteError(c, http.StatusUnsupportedMediaType, errInvalidCapabilityRequest)
		return
	}

	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxCapabilityRequestBytes)
	decoder := json.NewDecoder(c.Request.Body)
	decoder.DisallowUnknownFields()
	var payload capabilityRequest
	if err := decoder.Decode(&payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, errInvalidCapabilityRequest)
		return
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		utils.WriteError(c, http.StatusBadRequest, errInvalidCapabilityRequest)
		return
	}

	requestID, err := uuid.Parse(payload.RequestID)
	if err != nil || requestID.Version() != 4 || requestID.Variant() != uuid.RFC4122 ||
		!SupportedContentType(payload.ContentType) {
		utils.WriteError(c, http.StatusBadRequest, errInvalidCapabilityRequest)
		return
	}

	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidToken)
		return
	}
	granted, err := h.store.HasReceiptOCRGrant(c.Request.Context(), userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if !granted {
		utils.WriteError(c, http.StatusForbidden, errReceiptOCRNotGranted)
		return
	}

	normalizedType := strings.ToLower(strings.TrimSpace(payload.ContentType))
	token, expiresAt, err := MintCapability(
		h.secret,
		h.now(),
		userID,
		requestID.String(),
		normalizedType,
	)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	c.Header("Cache-Control", "no-store")
	utils.WriteJSON(c, http.StatusCreated, CapabilityResponse{
		Token:       token,
		RequestID:   requestID.String(),
		ContentType: normalizedType,
		MaxBytes:    MaxDocumentBytes,
		ExpiresAt:   expiresAt,
	})
}
