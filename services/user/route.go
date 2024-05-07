package user

import (
	"expense-tracker/services/auth"
	"expense-tracker/types"
	"expense-tracker/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	store types.UserStore
}

func NewHandler(store types.UserStore) *Handler {
	return &Handler{
		store: store,
	}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/register", h.handleRegister)
	router.POST("/login", h.handleLogin)
}

func (h *Handler) handleRegister(c *gin.Context) {
	// get json payload
	var payload types.RegisterUserPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	// check if the user email exist
	_, err := h.store.GetUserByEmail(payload.Email)
	if err == nil {
		utils.WriteError(c, http.StatusBadRequest,
			fmt.Errorf("email %s already exists.", payload.Email))
		return
	}

	// check if the username exist
	_, err = h.store.GetUserByUsername(payload.Username)
	if err == nil {
		utils.WriteError(c, http.StatusBadRequest,
			fmt.Errorf("username %s already exists.", payload.Email))
		return
	}

	// if not, create new user
	hashedPassword, err := auth.HashPassword(payload.Password)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	err = h.store.CreateUser(types.User{
		ID:             uuid.New(),
		Username:       payload.Username,
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

	utils.ParseJSON(c, nil)
}

func (h *Handler) handleLogin(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "login handler",
	})
}
