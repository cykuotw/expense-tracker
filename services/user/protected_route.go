package user

import (
	"expense-tracker/services/auth"
	"expense-tracker/types"
	"expense-tracker/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

type HandlerProtected struct {
	store types.UserStore
}

func NewProtectedHandler(store types.UserStore) *HandlerProtected {
	return &HandlerProtected{
		store: store,
	}
}

func (h *HandlerProtected) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/user_info", h.handleUserInfo)
}

func (h *HandlerProtected) handleUserInfo(c *gin.Context) {
	// get user id from jwt
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// request user info
	user, err := h.store.GetUserByID(userID)

	response := types.UserInfoResponse{
		Nickname:  user.Nickname,
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
		Email:     user.Email,
	}
	utils.WriteJSON(c, http.StatusOK, response)
}
