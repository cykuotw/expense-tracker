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
	public.GET("/invitations/:token", h.handleGetInvitation)

	adminProtected.POST("/invitations", h.handleCreateInvitation)
	adminProtected.GET("/invitations", h.handleListInvitations)
	adminProtected.POST("/invitations/:token/expire", h.handleExpireInvitation)
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
		utils.WriteError(c, http.StatusBadRequest,
			fmt.Errorf("invalid payload %v", errors))
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

	token := uuid.NewString()
	now := time.Now()
	expiresAt := now.Add(24 * time.Hour) // 1 days

	invitation := types.Invitation{
		ID:        uuid.New(),
		Token:     token,
		Email:     payload.Email,
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

func (h *Handler) handleGetInvitation(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.WriteError(c, http.StatusBadRequest, fmt.Errorf("token is required"))
		return
	}

	invitation, err := h.store.GetInvitationByToken(token)
	if err != nil {
		utils.WriteError(c, http.StatusNotFound, fmt.Errorf("invitation not found"))
		return
	}

	if invitation.UsedAt != nil {
		utils.WriteError(c, http.StatusBadRequest, fmt.Errorf("invitation already used"))
		return
	}

	if time.Now().After(invitation.ExpiresAt) {
		utils.WriteError(c, http.StatusBadRequest, fmt.Errorf("invitation expired"))
		return
	}

	utils.WriteJSON(c, http.StatusOK, types.InvitationResponse{
		Email: invitation.Email,
		Valid: true,
	})
}

func (h *Handler) handleListInvitations(c *gin.Context) {
	invitations, err := h.store.GetInvitations()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusOK, invitations)
}

func (h *Handler) handleExpireInvitation(c *gin.Context) {
	token := c.Param("token")
	if token == "" {
		utils.WriteError(c, http.StatusBadRequest, fmt.Errorf("token is required"))
		return
	}

	if err := h.store.ExpireInvitation(token); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusOK, nil)
}
