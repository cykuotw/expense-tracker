package notification

import (
	"context"
	"encoding/base64"
	"errors"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net"
	"net/http"
	"net/netip"
	"net/url"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	store     *Store
	publicKey string
}

func NewHandler(store *Store, publicKey string) *Handler {
	return &Handler{store: store, publicKey: strings.TrimSpace(publicKey)}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/notifications/settings", h.handleSettings)
	router.POST("/notifications/subscriptions", h.handleCreateSubscription)
	router.DELETE("/notifications/subscriptions", h.handleDeleteSubscription)
	router.PATCH("/notifications/subscriptions/details", h.handleDetails)
	router.PUT("/notifications/groups/:groupID/mute", h.handleGroupMute)
}

func notificationUserID(c *gin.Context) (uuid.UUID, bool) {
	value, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err)
		return uuid.Nil, false
	}
	userID, err := uuid.Parse(value)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidJWTToken)
		return uuid.Nil, false
	}
	return userID, true
}

func (h *Handler) handleSettings(c *gin.Context) {
	userID, ok := notificationUserID(c)
	if !ok {
		return
	}
	settings, err := h.store.Settings(c.Request.Context(), userID, h.publicKey)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, settings)
}

func (h *Handler) handleCreateSubscription(c *gin.Context) {
	if h.publicKey == "" {
		utils.WriteError(c, http.StatusConflict, types.ErrWebPushUnavailable)
		return
	}
	userID, ok := notificationUserID(c)
	if !ok {
		return
	}
	var input types.WebPushSubscriptionInput
	if err := utils.ParseJSON(c, &input); err != nil || validateSubscription(c.Request.Context(), input) != nil {
		utils.WriteError(c, http.StatusBadRequest, types.ErrInvalidWebPushSubscription)
		return
	}
	if err := h.store.UpsertSubscription(c.Request.Context(), userID, input); err != nil {
		writeNotificationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) handleDeleteSubscription(c *gin.Context) {
	userID, ok := notificationUserID(c)
	if !ok {
		return
	}
	var payload struct {
		Endpoint string `json:"endpoint"`
	}
	if err := utils.ParseJSON(c, &payload); err != nil || strings.TrimSpace(payload.Endpoint) == "" {
		utils.WriteError(c, http.StatusBadRequest, types.ErrInvalidWebPushSubscription)
		return
	}
	if err := h.store.DeleteSubscription(c.Request.Context(), userID, payload.Endpoint); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) handleDetails(c *gin.Context) {
	userID, ok := notificationUserID(c)
	if !ok {
		return
	}
	var payload struct {
		Endpoint    string `json:"endpoint"`
		ShowDetails bool   `json:"showDetails"`
	}
	if err := utils.ParseJSON(c, &payload); err != nil || strings.TrimSpace(payload.Endpoint) == "" {
		utils.WriteError(c, http.StatusBadRequest, types.ErrInvalidWebPushSubscription)
		return
	}
	if err := h.store.SetDetails(c.Request.Context(), userID, payload.Endpoint, payload.ShowDetails); err != nil {
		writeNotificationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h *Handler) handleGroupMute(c *gin.Context) {
	userID, ok := notificationUserID(c)
	if !ok {
		return
	}
	groupID, err := uuid.Parse(c.Param("groupID"))
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, types.ErrGroupNotExist)
		return
	}
	var payload struct {
		Muted bool `json:"muted"`
	}
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	if err := h.store.SetGroupMute(c.Request.Context(), userID, groupID, payload.Muted); err != nil {
		writeNotificationError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func writeNotificationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, types.ErrPermissionDenied):
		utils.WriteError(c, http.StatusForbidden, err)
	case errors.Is(err, types.ErrWebPushSubscriptionLimit), errors.Is(err, types.ErrWebPushUnavailable):
		utils.WriteError(c, http.StatusConflict, err)
	default:
		utils.WriteError(c, http.StatusInternalServerError, err)
	}
}

func validateSubscription(ctx context.Context, input types.WebPushSubscriptionInput) error {
	parsed, err := url.Parse(strings.TrimSpace(input.Endpoint))
	if err != nil || parsed.Scheme != "https" || parsed.User != nil || parsed.Hostname() == "" || parsed.Port() != "" && parsed.Port() != "443" || !supportedPushHost(parsed.Hostname()) {
		return types.ErrInvalidWebPushSubscription
	}
	if err := ensurePublicHost(ctx, parsed.Hostname()); err != nil {
		return types.ErrInvalidWebPushSubscription
	}
	p256dh, err := base64.RawURLEncoding.DecodeString(input.P256DH)
	if err != nil || len(p256dh) != 65 || p256dh[0] != 4 {
		return types.ErrInvalidWebPushSubscription
	}
	authKey, err := base64.RawURLEncoding.DecodeString(input.Auth)
	if err != nil || len(authKey) != 16 {
		return types.ErrInvalidWebPushSubscription
	}
	return nil
}

func supportedPushHost(host string) bool {
	host = strings.ToLower(strings.TrimSuffix(host, "."))
	return host == "fcm.googleapis.com" || host == "updates.push.services.mozilla.com" || host == "web.push.apple.com"
}

func ensurePublicHost(ctx context.Context, host string) error {
	lookupCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	addresses, err := net.DefaultResolver.LookupNetIP(lookupCtx, "ip", host)
	if err != nil || len(addresses) == 0 {
		return types.ErrInvalidWebPushSubscription
	}
	for _, address := range addresses {
		if !isPublicAddress(address) {
			return types.ErrInvalidWebPushSubscription
		}
	}
	return nil
}

func isPublicAddress(address netip.Addr) bool {
	return address.IsValid() && !address.IsPrivate() && !address.IsLoopback() &&
		!address.IsLinkLocalUnicast() && !address.IsLinkLocalMulticast() &&
		!address.IsMulticast() && !address.IsUnspecified()
}
