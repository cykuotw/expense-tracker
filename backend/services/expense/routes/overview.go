package expense

import (
	"errors"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) handleGetGroupOverview(c *gin.Context) {
	groupID := c.Param("groupId")
	page, err := strconv.ParseInt(c.Param("page"), 10, 0)
	if groupID == "" || err != nil || page < 0 {
		utils.WriteError(c, http.StatusBadRequest, types.ErrGroupNotExist)
		return
	}
	order := types.ExpenseListOrder(c.DefaultQuery("order", string(types.ExpenseListOrderNewest)))
	status := types.ExpenseListStatus(c.DefaultQuery("status", string(types.ExpenseListStatusUnsettled)))
	if (order != types.ExpenseListOrderNewest && order != types.ExpenseListOrderOldest) ||
		(status != types.ExpenseListStatusAll && status != types.ExpenseListStatusUnsettled && status != types.ExpenseListStatusSettled) {
		utils.WriteError(c, http.StatusBadRequest, errors.New("invalid expense list filters"))
		return
	}

	userID := c.GetString("userID")
	group, err := h.groupStore.GetGroupByID(groupID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	members, err := h.groupStore.GetGroupMemberByGroupID(groupID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	balances, err := h.balanceResponse(groupID, userID, group.Currency)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	expenses, err := h.expenseListResponse(groupID, page, order, status, userID, group.Currency)
	if err != nil {
		if errors.Is(err, types.ErrNoRemainingExpenses) {
			expenses = types.ExpenseResponsePage{Expenses: []types.ExpenseResponseBrief{}}
		} else {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
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
	utils.WriteJSON(c, http.StatusOK, types.GroupOverviewResponse{
		Group:    types.GetGroupResponse{GroupName: group.GroupName, Description: group.Description, Currency: group.Currency, GroupType: group.GroupType, Members: groupMembers},
		Balance:  balances,
		Expenses: expenses,
	})
}

func (h *Handler) balanceResponse(groupID, userID, currency string) (types.BalanceResponse, error) {
	items, err := h.store.GetBalanceByGroupId(groupID)
	if err != nil {
		return types.BalanceResponse{}, err
	}
	ids := make([]string, 0, len(items)*2)
	for _, item := range items {
		ids = append(ids, item.SenderUserID.String(), item.ReceiverUserID.String())
	}
	names, err := usernamesByIDs(h.userStore, ids)
	if err != nil {
		return types.BalanceResponse{}, err
	}
	balances := make([]types.BalanceRsp, 0, len(items))
	for _, item := range items {
		balances = append(balances, types.BalanceRsp{ID: item.ID, SenderUserID: item.SenderUserID, SenderUesrname: names[item.SenderUserID.String()], ReceiverUserID: item.ReceiverUserID, ReceiverUsername: names[item.ReceiverUserID.String()], Balance: item.Share})
	}
	return types.BalanceResponse{Currency: currency, CurrentUser: userID, Balances: balances}, nil
}

func (h *Handler) expenseListResponse(groupID string, page int64, order types.ExpenseListOrder, status types.ExpenseListStatus, userID, currency string) (types.ExpenseResponsePage, error) {
	pageData, err := h.store.GetExpenseList(groupID, page, order, status)
	if err != nil {
		return types.ExpenseResponsePage{}, err
	}
	expenseTypes, err := h.store.GetExpenseType()
	if err != nil {
		return types.ExpenseResponsePage{}, err
	}
	typesByID := make(map[uuid.UUID]*types.ExpenseType, len(expenseTypes))
	for _, item := range expenseTypes {
		typesByID[item.ID] = item
	}
	ids := make([]string, 0, len(pageData.Expenses))
	for _, item := range pageData.Expenses {
		ids = append(ids, item.ID.String())
	}
	ledgers, err := ledgersByExpenseIDs(h.store, ids)
	if err != nil {
		return types.ExpenseResponsePage{}, err
	}
	payerIDs := make([]string, 0)
	seen := make(map[string]struct{})
	for _, list := range ledgers {
		for _, ledger := range list {
			id := ledger.LenderUserID.String()
			if _, ok := seen[id]; !ok {
				seen[id] = struct{}{}
				payerIDs = append(payerIDs, id)
			}
		}
	}
	names, err := usernamesByIDs(h.userStore, payerIDs)
	if err != nil {
		return types.ExpenseResponsePage{}, err
	}
	result := make([]types.ExpenseResponseBrief, 0, len(pageData.Expenses))
	for _, item := range pageData.Expenses {
		payerIDs, payerNames := []uuid.UUID{}, []string{}
		inserted := make(map[string]struct{})
		for _, ledger := range ledgers[item.ID.String()] {
			id := ledger.LenderUserID.String()
			if _, ok := inserted[id]; !ok {
				inserted[id] = struct{}{}
				payerIDs = append(payerIDs, ledger.LenderUserID)
				payerNames = append(payerNames, names[id])
			}
		}
		response := types.ExpenseResponseBrief{ExpenseID: item.ID, Description: item.Description, Total: item.Total, ExpenseTime: item.ExpenseTime, CurrentUser: userID, Currency: currency, IsSettled: item.IsSettled, PayerUserIDs: payerIDs, PayerUsernames: payerNames, ExpenseTypeID: item.ExpenseTypeID}
		if expenseType := typesByID[item.ExpenseTypeID]; expenseType != nil {
			response.ExpenseType = expenseType.Name
			response.ExpenseCategory = expenseType.Category
		}
		result = append(result, response)
	}
	return types.ExpenseResponsePage{Expenses: result, HasMore: pageData.HasMore}, nil
}
