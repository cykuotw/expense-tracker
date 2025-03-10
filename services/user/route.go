package user

import (
	"expense-tracker/services/common"
	"expense-tracker/types"
	"expense-tracker/utils"
	"net/http"

	"github.com/gin-gonic/gin"
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
	router.POST("/checkEmail", common.Make(h.handleCheckEmail))
	router.POST("/userInfo", common.Make(h.handleGetUserInfoByEmail))
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

func (h *Handler) handleGetUserInfoByEmail(c *gin.Context) error {
	type emailRequest struct {
		Email string `json:"email"`
	}
	var payload emailRequest
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return nil
	}

	user, err := h.store.GetUserByEmail(payload.Email)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return nil
	}

	utils.WriteJSON(c, http.StatusOK, user)

	return nil
}
