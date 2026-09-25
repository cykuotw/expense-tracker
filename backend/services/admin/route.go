package admin

import (
	"context"
	"errors"
	"expense-tracker/backend/internal/observability"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type UserStore interface {
	GetAdminUsers() ([]types.AdminUserResponse, error)
	SetUserActive(actorID string, targetID string, active bool) error
	SetUserRole(actorID string, targetID string, role string) error
	SetReceiptOCRGrant(ctx context.Context, actorID string, targetID string, enabled bool) error
}

type InvitationStore interface {
	GetAdminInvitations() ([]types.AdminInvitationResponse, error)
	RotateInvitationTokenByID(id string) (string, error)
	ExpireInvitationByID(id string) error
}

type Handler struct {
	users       UserStore
	invitations InvitationStore
}

func NewHandler(users UserStore, invitations InvitationStore) *Handler {
	return &Handler{users: users, invitations: invitations}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/admin/users", h.handleList)
	router.PATCH("/admin/users/:id/status", h.handleUpdateStatus)
	router.PATCH("/admin/users/:id/role", h.handleUpdateRole)
	router.PATCH("/admin/users/:id/features/receipt-ocr", h.handleUpdateReceiptOCRGrant)
	router.POST("/admin/invitations/:id/link", h.handleInvitationLink)
	router.POST("/admin/invitations/:id/expire", h.handleExpireInvitation)
}

func (h *Handler) handleList(c *gin.Context) {
	users, err := h.users.GetAdminUsers()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	invitations, err := h.invitations.GetAdminInvitations()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if users == nil {
		users = []types.AdminUserResponse{}
	}
	if invitations == nil {
		invitations = []types.AdminInvitationResponse{}
	}
	utils.WriteJSON(c, http.StatusOK, gin.H{
		"users":       users,
		"invitations": invitations,
	})
}

func targetID(c *gin.Context) (string, error) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		return "", types.ErrInvalidAction
	}
	return id.String(), nil
}

func actorID(c *gin.Context) (string, error) {
	return auth.ExtractJWTClaim(c, "userID")
}

func writeMutationError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, types.ErrProtectedAdmin),
		errors.Is(err, types.ErrCannotDeactivateSelf),
		errors.Is(err, types.ErrCannotChangeOwnRole),
		errors.Is(err, types.ErrLastActiveAdmin),
		errors.Is(err, types.ErrAccountInactive):
		utils.WriteError(c, http.StatusConflict, err)
	case errors.Is(err, types.ErrPermissionDenied):
		utils.WriteError(c, http.StatusForbidden, err)
	case errors.Is(err, types.ErrUserNotExist):
		utils.WriteError(c, http.StatusNotFound, err)
	case errors.Is(err, types.ErrInvalidAction), errors.Is(err, types.ErrInvalidUserRole):
		utils.WriteError(c, http.StatusBadRequest, err)
	default:
		utils.WriteError(c, http.StatusInternalServerError, err)
	}
}

func (h *Handler) handleUpdateReceiptOCRGrant(c *gin.Context) {
	var payload types.UpdateReceiptOCRGrantPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	if err := utils.Validate.Struct(payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, utils.NewValidationError(err.(validator.ValidationErrors)))
		return
	}
	target, err := targetID(c)
	if err != nil {
		writeMutationError(c, err)
		return
	}
	actor, err := actorID(c)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err)
		return
	}

	if err := h.users.SetReceiptOCRGrant(c.Request.Context(), actor, target, *payload.Enabled); err != nil {
		logReceiptOCRGrantChange(c, actor, target, *payload.Enabled, grantMutationOutcome(err))
		writeMutationError(c, err)
		return
	}
	logReceiptOCRGrantChange(c, actor, target, *payload.Enabled, "updated")
	utils.WriteJSON(c, http.StatusOK, gin.H{
		"capabilities": types.UserCapabilities{ReceiptOCR: *payload.Enabled},
	})
}

func grantMutationOutcome(err error) string {
	if errors.Is(err, types.ErrUserNotExist) ||
		errors.Is(err, types.ErrAccountInactive) ||
		errors.Is(err, types.ErrPermissionDenied) {
		return "rejected"
	}
	return "failed"
}

func logReceiptOCRGrantChange(c *gin.Context, actorID, targetID string, enabled bool, outcome string) {
	observability.Logger(c).Info("user_feature_grant_change",
		slog.String("actor_id", actorID),
		slog.String("target_user_id", targetID),
		slog.String("feature", types.UserFeatureReceiptOCR),
		slog.Bool("enabled", enabled),
		slog.String("outcome", outcome),
	)
}

func (h *Handler) handleUpdateStatus(c *gin.Context) {
	var payload types.UpdateUserStatusPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	if err := utils.Validate.Struct(payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, utils.NewValidationError(err.(validator.ValidationErrors)))
		return
	}
	target, err := targetID(c)
	if err != nil {
		writeMutationError(c, err)
		return
	}
	actor, err := actorID(c)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err)
		return
	}
	if err := h.users.SetUserActive(actor, target, *payload.IsActive); err != nil {
		writeMutationError(c, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, gin.H{"isActive": *payload.IsActive})
}

func (h *Handler) handleUpdateRole(c *gin.Context) {
	var payload types.UpdateUserRolePayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	if err := utils.Validate.Struct(payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, utils.NewValidationError(err.(validator.ValidationErrors)))
		return
	}
	target, err := targetID(c)
	if err != nil {
		writeMutationError(c, err)
		return
	}
	actor, err := actorID(c)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, err)
		return
	}
	if err := h.users.SetUserRole(actor, target, payload.Role); err != nil {
		writeMutationError(c, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, gin.H{"role": payload.Role})
}

func (h *Handler) handleInvitationLink(c *gin.Context) {
	id, err := targetID(c)
	if err != nil {
		writeMutationError(c, err)
		return
	}
	token, err := h.invitations.RotateInvitationTokenByID(id)
	if err != nil {
		writeMutationError(c, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, gin.H{"token": token})
}

func (h *Handler) handleExpireInvitation(c *gin.Context) {
	id, err := targetID(c)
	if err != nil {
		writeMutationError(c, err)
		return
	}
	if err := h.invitations.ExpireInvitationByID(id); err != nil {
		writeMutationError(c, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, nil)
}
