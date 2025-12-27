package group

import (
	"expense-tracker/backend/types"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store     types.GroupStore
	userStore types.UserStore
}

func NewHandler(store types.GroupStore, userStore types.UserStore) *Handler {
	return &Handler{
		store:     store,
		userStore: userStore,
	}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/create_group", h.handleCreateGroup)
	router.GET("/group/:groupid", h.handleGetGroup)
	router.GET("/groups", h.handleGetGroupList)
	router.GET("/group_member/:groupid", h.handleGetGroupMember)
	router.PUT("/group_member", h.handleUpdateGroupMember)
	router.PUT("/archive_group/:groupId", h.handleArchiveGroup)

	router.GET("/related_member", h.handleGetRelatedMember)
}
