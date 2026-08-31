package route

import (
	"errors"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func (h *Handler) handleInvitationExchange(c *gin.Context) {
	var payload types.ExchangeInvitationPayload
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, 1024)
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	payload.Token = strings.TrimSpace(payload.Token)
	if err := utils.Validate.Struct(payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, utils.NewValidationError(err.(validator.ValidationErrors)))
		return
	}

	session, err := auth.GenerateOpaqueToken()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	invitation, err := h.invitationStore.ExchangeInvitation(payload.Token, session)
	if err != nil {
		if errors.Is(err, types.ErrInvitationInvalid) {
			utils.WriteError(c, http.StatusForbidden, err)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	setRegistrationSessionCookie(c, session)
	utils.WriteJSON(c, http.StatusOK, types.InvitationResponse{Email: invitation.Email, Valid: true})
}
