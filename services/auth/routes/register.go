package route

import (
	"expense-tracker/services/auth"
	"expense-tracker/types"
	"expense-tracker/utils"
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

	// check if the user email exist
	_, err := h.store.GetUserByEmail(payload.Email)
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
	})
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}
