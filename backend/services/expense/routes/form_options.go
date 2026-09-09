package expense

import (
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
)

func expenseTypeResponses(expenseTypes []*types.ExpenseType) []types.ExpenseTypeResponse {
	response := make([]types.ExpenseTypeResponse, 0, len(expenseTypes))
	for _, expenseType := range expenseTypes {
		response = append(response, types.ExpenseTypeResponse{
			ID:       expenseType.ID.String(),
			Category: expenseType.Category,
			Name:     expenseType.Name,
		})
	}
	return response
}

func (h *Handler) expenseFormGroup(groupID, userID string) (types.GetGroupResponse, error) {
	group, err := h.groupStore.GetGroupByIDAndUser(groupID, userID)
	if err != nil {
		return types.GetGroupResponse{}, err
	}
	members, err := h.groupStore.GetGroupMemberByGroupID(groupID)
	if err != nil {
		return types.GetGroupResponse{}, err
	}
	currencyEditable, err := h.groupStore.CanEditGroupCurrency(groupID, userID)
	if err != nil {
		return types.GetGroupResponse{}, err
	}

	groupMembers := make([]types.GroupMember, 0, len(members))
	var current *types.GroupMember
	for _, member := range members {
		item := types.GroupMember{UserID: member.ID.String(), Username: member.Username}
		if item.UserID == userID {
			current = &item
		} else {
			groupMembers = append(groupMembers, item)
		}
	}
	if current != nil {
		groupMembers = append(groupMembers, *current)
	}

	return types.GetGroupResponse{
		GroupName:        group.GroupName,
		Description:      group.Description,
		Currency:         group.Currency,
		CurrencyEditable: currencyEditable,
		DetailsEditable:  group.CreateByUser.String() == userID,
		GroupType:        group.GroupType,
		Members:          groupMembers,
	}, nil
}

func (h *Handler) handleGetCreateExpenseOptions(c *gin.Context) {
	userID := c.GetString("userID")
	groups, err := h.groupStore.GetGroupListByUser(userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	expenseTypes, err := h.store.GetExpenseType()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	currencies, err := h.groupStore.ListCurrencies()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	var groupResponse *types.GetGroupResponse
	if groupID := c.Query("groupId"); groupID != "" {
		group, groupErr := h.expenseFormGroup(groupID, userID)
		if groupErr != nil {
			utils.WriteError(c, http.StatusNotFound, types.ErrGroupNotExist)
			return
		}
		groupResponse = &group
	}

	utils.WriteJSON(c, http.StatusOK, types.CreateExpenseOptionsResponse{
		Groups:       groups,
		ExpenseTypes: expenseTypeResponses(expenseTypes),
		Currencies:   currencies,
		Group:        groupResponse,
	})
}

func (h *Handler) handleGetEditExpenseOptions(c *gin.Context) {
	userID := c.GetString("userID")
	expenseTypes, err := h.store.GetExpenseType()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	currencies, err := h.groupStore.ListCurrencies()
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	expenseResponse, err := h.expenseDetailResponse(c, expenseTypes)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	groups, err := h.groupStore.GetGroupListByUser(userID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	groupID := c.DefaultQuery("groupId", expenseResponse.GroupId)
	group, err := h.expenseFormGroup(groupID, userID)
	if err != nil {
		utils.WriteError(c, http.StatusNotFound, types.ErrGroupNotExist)
		return
	}

	utils.WriteJSON(c, http.StatusOK, types.EditExpenseOptionsResponse{
		Expense:      expenseResponse,
		Groups:       groups,
		ExpenseTypes: expenseTypeResponses(expenseTypes),
		Currencies:   currencies,
		Group:        group,
	})
}
