package route

import (
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

func (h *Handler) handleRegister(c *gin.Context) {
	// get json payload
	var payload types.RegisterUserPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	if err := utils.Validate.Struct(payload); err != nil {
		errors := err.(validator.ValidationErrors)
		utils.WriteError(c, http.StatusBadRequest,
			fmt.Errorf("invalid payload %v", errors))
		return
	}

	// validate invitation token
	invitation, err := h.invitationStore.GetInvitationByToken(payload.Token)
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, fmt.Errorf("invalid invitation token"))
		return
	}

	if invitation.UsedAt != nil {
		utils.WriteError(c, http.StatusUnauthorized, fmt.Errorf("invitation token already used"))
		return
	}

	if time.Now().After(invitation.ExpiresAt) {
		utils.WriteError(c, http.StatusUnauthorized, fmt.Errorf("invitation token expired"))
		return
	}

	if invitation.Email != "" && invitation.Email != payload.Email {
		utils.WriteError(c, http.StatusBadRequest, fmt.Errorf("email does not match invitation"))
		return
	}

	// check if the user email exist
	_, err = h.store.GetUserByEmail(payload.Email)
	if err == nil {
		utils.WriteError(c, http.StatusBadRequest,
			fmt.Errorf("email %s already exists.", payload.Email))
		return
	}

	// if not, create new user
	hashedPassword, err := auth.HashPassword(payload.Password)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	username := payload.Nickname
	if username == "" {
		username = payload.Firstname + " " + payload.Lastname
	}

	err = h.store.CreateUser(types.User{
		ID:             uuid.New(),
		Username:       username,
		Nickname:       payload.Nickname,
		Firstname:      payload.Firstname,
		Lastname:       payload.Lastname,
		Email:          payload.Email,
		PasswordHashed: hashedPassword,
		ExternalType:   "",
		ExternalID:     "",
		CreateTime:     time.Now(),
		IsActive:       true,
		Role:           "user",
	})
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	if err := h.invitationStore.MarkInvitationUsed(payload.Token, payload.Email); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}
