package route

import (
	"errors"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
	"github.com/google/uuid"
)

func (h *Handler) handleRegister(c *gin.Context) {
	var payload types.RegisterUserPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	payload.Email = auth.NormalizeEmail(payload.Email)
	payload.Token = strings.TrimSpace(payload.Token)
	if err := utils.Validate.Struct(payload); err != nil {
		validationErrors := err.(validator.ValidationErrors)
		if len(validationErrors) == 1 && hasValidationError(validationErrors, "Token", "required") {
			utils.WriteError(c, http.StatusBadRequest, types.ErrInvitationRequired)
			return
		}
		utils.WriteError(c, http.StatusBadRequest, utils.NewValidationError(validationErrors))
		return
	}

	// Password hashing intentionally happens before the transaction acquires the invitation lock.
	hashedPassword, err := auth.HashPassword(payload.Password)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	username := strings.TrimSpace(payload.Nickname)
	if username == "" {
		username = strings.TrimSpace(payload.Firstname + " " + payload.Lastname)
	}

	user := types.User{
		ID:               uuid.New(),
		Username:         username,
		Nickname:         strings.TrimSpace(payload.Nickname),
		Firstname:        strings.TrimSpace(payload.Firstname),
		Lastname:         strings.TrimSpace(payload.Lastname),
		Email:            payload.Email,
		PasswordHashed:   hashedPassword,
		HasLocalPassword: true,
		ExternalType:     "",
		ExternalID:       "",
		CreateTime:       time.Now(),
		IsActive:         true,
		Role:             "user",
	}

	if h.registrationStore == nil {
		utils.WriteError(c, http.StatusInternalServerError, errors.New("registration store is unavailable"))
		return
	}
	if err := h.registrationStore.CreateInvitedUser(c.Request.Context(), payload.Token, user); err != nil {
		writeRegistrationError(c, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}

func hasValidationError(validationErrors validator.ValidationErrors, field string, tag string) bool {
	for _, validationError := range validationErrors {
		if validationError.Field() == field && validationError.Tag() == tag {
			return true
		}
	}
	return false
}

func registrationStatus(err error) int {
	switch {
	case errors.Is(err, types.ErrInvitationRequired):
		return http.StatusBadRequest
	case errors.Is(err, types.ErrInvitationInvalid),
		errors.Is(err, types.ErrInvitationExpired),
		errors.Is(err, types.ErrInvitationUsed),
		errors.Is(err, types.ErrInvitationEmailMismatch):
		return http.StatusForbidden
	case errors.Is(err, types.ErrAccountConflict), errors.Is(err, types.ErrGoogleAccountConflict):
		return http.StatusConflict
	default:
		return http.StatusInternalServerError
	}
}

func writeRegistrationError(c *gin.Context, err error) error {
	status := registrationStatus(err)
	utils.WriteError(c, status, err)
	if status >= http.StatusInternalServerError {
		return errors.New("registration failed")
	}
	return err
}
