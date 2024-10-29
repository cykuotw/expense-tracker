package user

import (
	"expense-tracker/config"
	"expense-tracker/services/auth"
	"expense-tracker/services/common"
	"expense-tracker/types"
	"expense-tracker/utils"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-playground/validator/v10"
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
	router.POST("/auth", h.handle3rdParty)

	router.POST("/checkEmail", common.Make(h.handleCheckEmail))
}

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

func (h *Handler) handleLogin(c *gin.Context) {
	// get json payload
	var payload types.LoginUserPayload
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

	var user *types.User
	var err error
	user, err = h.store.GetUserByEmail(payload.Email)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	if !auth.ValidatePassword(user.PasswordHashed, payload.Password) {
		utils.WriteError(c, http.StatusBadRequest, types.ErrPasswordNotMatch)
		return
	}

	secret := []byte(config.Envs.JWTSecret)
	token, err := auth.CreateJWT(secret, user.ID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, map[string]string{"token": token})
}

func (h *Handler) handle3rdParty(c *gin.Context) {
	var payload types.ThirdPartyUserPayload

	if err := utils.ParseJSON(c, &payload); err != nil {
		fmt.Println(err)
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	succeed := false
	user, err := h.store.GetUserByEmail(payload.Email)
	if err == types.ErrUserNotExist {
		// register
		hashedPassword, err := auth.HashPassword(payload.ExternalId)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		user = &types.User{
			ID:             uuid.New(),
			Username:       payload.Nickname,
			Nickname:       payload.Nickname,
			Firstname:      payload.Firstname,
			Lastname:       payload.Lastname,
			Email:          payload.Email,
			PasswordHashed: hashedPassword,
			ExternalType:   payload.ExternalType,
			ExternalID:     payload.ExternalId,
			CreateTime:     time.Now(),
			IsActive:       true,
		}
		err = h.store.CreateUser(*user)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}

		succeed = true
	} else if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	} else {
		// login
		if user.ExternalID != payload.ExternalId {
			utils.WriteError(c, http.StatusBadRequest, types.ErrPasswordNotMatch)
			return
		}
		if !auth.ValidatePassword(user.PasswordHashed, payload.ExternalId) {
			utils.WriteError(c, http.StatusBadRequest, types.ErrPasswordNotMatch)
			return
		}

		succeed = true
	}

	if succeed {
		secret := []byte(config.Envs.JWTSecret)
		token, err := auth.CreateJWT(secret, user.ID)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		utils.WriteJSON(c, http.StatusOK, map[string]string{"token": token})
		return
	}
	utils.WriteError(c, http.StatusInternalServerError, nil)
}

func (h *Handler) handleCheckEmail(c *gin.Context) error {
	type emailRequest struct {
		Email string `json:"email"`
	}
	var payload emailRequest
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return nil
	}

	exist, err := h.store.CheckEmailExist(payload.Email)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return nil
	}

	utils.WriteJSON(c, http.StatusOK, map[string]bool{"exist": exist})

	return nil
}
