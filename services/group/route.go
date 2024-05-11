package group

import (
	"expense-tracker/services/auth"
	"expense-tracker/types"
	"expense-tracker/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
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
	router.GET("/group/:groupID", h.handleGetGroup)
	router.GET("/group_list", h.handleGetGroupList)
	router.PUT("/group_member/:action/:userId", h.handleUpdateGroupMember)
	router.PUT("/archive_group/:groupId", h.handleArchiveGroup)
}

func (h *Handler) handleCreateGroup(c *gin.Context) {
	// get payload
	var payload types.CreateGroupPayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	// get user id from jwt
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// check if user id exist
	user, err := h.userStore.GetUserByID(userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	if payload.GroupName == "" {
		payload.GroupName = "Default Group Name"
	}

	group := types.Group{
		ID:           uuid.New(),
		GroupName:    payload.GroupName,
		Description:  payload.Description,
		CreateTime:   time.Now(),
		IsActive:     true,
		CreateByUser: user.ID,
	}

	err = h.store.CreateGroup(group)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, map[string]string{"groupId": group.ID.String()})
}

func (h *Handler) handleGetGroup(c *gin.Context) {
	// get group id
	groupId := c.Param("groupID")
	if groupId == "" {
		utils.WriteError(c, http.StatusBadRequest, types.ErrGroupNotExist)
		return
	}

	// get group detail by id
	group, err := h.store.GetGroupByID(groupId)
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	// get members of the group
	users, err := h.store.GetGroupMemberByGroupID(groupId)
	if err != nil {
		utils.WriteJSON(c, http.StatusInternalServerError, err)
		return
	}

	// make response
	var members []types.GroupMember
	for _, user := range users {
		member := types.GroupMember{
			UserID:   user.ID.String(),
			Username: user.Username,
		}
		members = append(members, member)
	}

	response := types.GetGroupResponse{
		GroupName:   group.GroupName,
		Description: group.Description,
		Members:     members,
	}

	utils.WriteJSON(c, http.StatusOK, response)
}

func (h *Handler) handleGetGroupList(c *gin.Context) {
	// get user id from jwt
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// get group id list where user id as member
	groups, err := h.store.GetGroupListByUser(userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// make response
	var response []types.GetGroupListResponse
	for _, group := range groups {
		res := types.GetGroupListResponse{
			ID:          group.ID.String(),
			GroupName:   group.GroupName,
			Description: group.Description,
		}
		response = append(response, res)
	}

	utils.WriteJSON(c, http.StatusOK, response)
}

func (h *Handler) handleUpdateGroupMember(c *gin.Context) {
	// get param from path
	action := c.Param("action")
	userId := c.Param("userId")

	if action != "add" && action != "delete" {
		utils.WriteError(c, http.StatusBadRequest, types.ErrInvalidAction)
		return
	}
	_, err := h.userStore.GetUserByID(userId)
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, types.ErrUserNotExist)
		return
	}

	// update group member
	if err := h.store.UpdateGroupMember(action, userId); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}

func (h *Handler) handleArchiveGroup(c *gin.Context) {
	// get param from path
	groupID := c.Param("groupId")
	_, err := h.store.GetGroupByID(groupID)
	if err != nil {
		utils.WriteJSON(c, http.StatusBadRequest, err)
		return
	}

	// update group status
	if err = h.store.UpdateGroupStatus(groupID, false); err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}
