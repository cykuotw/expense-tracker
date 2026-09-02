package invitation

import (
	"errors"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

type Handler struct {
	store types.InvitationStore
}

func NewHandler(store types.InvitationStore) *Handler {
	return &Handler{store: store}
}

func (h *Handler) RegisterRoutes(public *gin.RouterGroup, adminProtected *gin.RouterGroup) {
	adminProtected.POST("/invitations", h.handleCreateInvitation)
}

func (h *Handler) handleCreateInvitation(c *gin.Context) {
	var payload types.CreateInvitationPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		if !errors.Is(err, types.ErrEmptyRequestBody) {
			utils.WriteError(c, http.StatusBadRequest, err)
			return
		}
	}

	if err := utils.Validate.Struct(payload); err != nil {
		errors := err.(validator.ValidationErrors)
		utils.WriteError(c, http.StatusBadRequest, utils.NewValidationError(errors))
		return
	}

	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, fmt.Errorf("unauthorized"))
		return
	}

	inviterID, err := uuid.Parse(userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	token, err := auth.GenerateOpaqueToken()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour) // 1 days

	invitation := types.Invitation{
		ID:        uuid.New(),
		TokenHash: auth.HashToken(token),
		Email:     auth.NormalizeEmail(payload.Email),
		InviterID: inviterID,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}

	if err := h.store.CreateInvitation(invitation); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, gin.H{"token": token})
}
