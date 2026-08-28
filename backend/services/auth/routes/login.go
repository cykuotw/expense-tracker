package route

import (
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
)

func (h *Handler) handleLogin(c *gin.Context) {
	// get json payload
	var payload types.LoginUserPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	if err := utils.Validate.Struct(payload); err != nil {
		errors := err.(validator.ValidationErrors)
		utils.WriteError(c, http.StatusBadRequest, utils.NewValidationError(errors))
		return
	}
	payload.Email = auth.NormalizeEmail(payload.Email)

	var user *types.User
	var err error
	user, err = h.store.GetUserByEmail(payload.Email)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	if !user.HasLocalPassword {
		utils.WriteError(c, http.StatusBadRequest, types.ErrPasswordNotMatch)
		return
	}

	if !auth.ValidatePassword(user.PasswordHashed, payload.Password) {
		utils.WriteError(c, http.StatusBadRequest, types.ErrPasswordNotMatch)
		return
	}
	if !user.IsActive {
		utils.WriteError(c, http.StatusForbidden, types.ErrAccountInactive)
		return
	}

	if err := h.issueAuthSession(c, user); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, nil)
}
