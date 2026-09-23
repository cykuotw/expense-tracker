package group

import (
	"expense-tracker/backend/services/exchangerate"
	"expense-tracker/backend/types"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	store      types.GroupStore
	userStore  types.UserStore
	rateClient exchangerate.Client
}

func NewHandler(store types.GroupStore, userStore types.UserStore) *Handler {
	return &Handler{
		store:      store,
		userStore:  userStore,
		rateClient: exchangerate.NewFrankfurterClient(),
	}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/create_group", h.handleCreateGroup)
	router.GET("/currencies", h.handleListCurrencies)
	router.POST("/currency-recommendations", h.handleCurrencyRecommendations)
	router.GET("/group/:groupid", h.handleGetGroup)
	router.PUT("/group/:groupid", h.handleUpdateGroup)
	router.PUT("/group/:groupid/currency", h.handleUpdateGroupCurrency)
	router.GET("/group/:groupid/currency-settings", h.handleGetGroupCurrencySettings)
	router.PUT("/group/:groupid/currency-settings", h.handleUpdateGroupCurrencySettings)
	router.GET("/groups", h.handleGetGroupList)
	router.GET("/group_member/:groupid", h.handleGetGroupMember)
	router.PUT("/group_member", h.handleUpdateGroupMember)
	router.PUT("/group_members", h.handleReplaceGroupMembers)
	router.PUT("/archive_group/:groupId", h.handleArchiveGroup)

	router.GET("/related_member", h.handleGetRelatedMember)
}
