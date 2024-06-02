package expense

import (
	"expense-tracker/services/auth"
	"expense-tracker/types"
	"expense-tracker/utils"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type Handler struct {
	store      types.ExpenseStore
	userStore  types.UserStore
	groupStore types.GroupStore
}

func NewHandler(store types.ExpenseStore, userStore types.UserStore, groupStore types.GroupStore) *Handler {
	return &Handler{
		store:      store,
		userStore:  userStore,
		groupStore: groupStore,
	}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/create_expense", h.handleCreateExpense)
	router.GET("/expense_list/:groupId", h.handleGetExpenseList)
	router.GET("/expense_list/:groupId/:page", h.handleGetExpenseList)
	router.GET("/expense/:expenseId", h.handleGetExpenseDetail)
	router.PUT("/expense/:expenseId", h.handleUpdateExpense)
	router.PUT("/settle_expense/:groupId", h.handleSettleExpense)
	router.GET("/balance/:groupId", h.handleGetUnsettledBalance)
}

func (h *Handler) handleCreateExpense(c *gin.Context) {
	// extract payload
	var payload types.ExpensePayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	// extract jwt claim
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	if userID != payload.CreateByUserID {
		utils.WriteError(c, http.StatusForbidden, types.ErrUserNotPermitted)
		return
	}

	// check payload group id valid & user id valid
	_, err = h.groupStore.GetGroupByIDAndUser(payload.GroupID, payload.CreateByUserID)
	if err == types.ErrGroupNotExist || err == types.ErrUserNotExist {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	} else if err == types.ErrUserNotPermitted {
		utils.WriteError(c, http.StatusForbidden, err)
		return
	} else if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// create expense
	expenseID := uuid.New()
	groupID, err := uuid.Parse(payload.GroupID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	creatorID, err := uuid.Parse(payload.CreateByUserID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	payerID, err := uuid.Parse(payload.PayByUserId)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	expTypeID, err := uuid.Parse(payload.ExpenseTypeID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	expense := types.Expense{
		ID:             expenseID,
		Description:    payload.Description,
		GroupID:        groupID,
		CreateByUserID: creatorID,
		PayByUserId:    payerID,
		ExpenseTypeID:  expTypeID,
		ProviderName:   payload.ProviderName,
		IsSettled:      false,
		SubTotal:       payload.SubTotal,
		TaxFeeTip:      payload.TaxFeeTip,
		Total:          payload.Total,
		Currency:       payload.Currency,
		InvoicePicUrl:  payload.InvoicePicUrl,
	}

	err = h.store.CreateExpense(expense)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// create items
	for _, itemPayload := range payload.Items {
		// create provider if not exist
		item := types.Item{
			ID:        uuid.New(),
			ExpenseID: expenseID,
			Name:      itemPayload.ItemName,
			Amount:    itemPayload.Amount,
			Unit:      itemPayload.Unit,
			UnitPrice: itemPayload.UnitPrice,
		}
		err = h.store.CreateItem(item)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
	}

	// create ledgers
	for _, ledgerPayload := range payload.Ledgers {
		lenderUserId, err := uuid.Parse(ledgerPayload.LenderUserID)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		borrowerUserId, err := uuid.Parse(ledgerPayload.BorrowerUesrID)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		ledger := types.Ledger{
			ID:             uuid.New(),
			ExpenseID:      expenseID,
			LenderUserID:   lenderUserId,
			BorrowerUesrID: borrowerUserId,
			Share:          ledgerPayload.Share,
		}

		err = h.store.CreateLedger(ledger)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
	}

	utils.WriteJSON(c, http.StatusCreated, map[string]string{"expenseId": expenseID.String()})
}

func (h *Handler) handleGetExpenseList(c *gin.Context) {
	// get group id, page from param
	groupIdStr := c.Param("groupId")
	if groupIdStr == "" {
		utils.WriteError(c, http.StatusBadRequest, types.ErrGroupNotExist)
		return
	}

	pageStr := c.Param("page")
	var err error
	page := int64(0)
	if pageStr != "" {
		page, err = strconv.ParseInt(pageStr, 10, 0)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
	}

	// extract user id from jwt claim, and check permission
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	_, err = h.groupStore.GetGroupByIDAndUser(groupIdStr, userID)
	if err == types.ErrGroupNotExist || err == types.ErrUserNotExist {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	} else if err == types.ErrUserNotPermitted {
		utils.WriteError(c, http.StatusForbidden, err)
		return
	} else if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// get expense list wrt page
	expenseList, err := h.store.GetExpenseList(groupIdStr, page)
	if err == types.ErrNoRemainingExpenses {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	} else if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	var response []types.ExpenseResponseBrief
	for _, expense := range expenseList {
		var payerUserIDs []uuid.UUID
		var payerUsernames []string

		ledgers, err := h.store.GetLedgersByExpenseID(expense.ID.String())
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}

		inserted := make(map[string]interface{})
		for _, ledger := range ledgers {
			_, ok := inserted[ledger.ID.String()]
			if !ok {
				payerUserIDs = append(payerUserIDs, ledger.LenderUserID)
				user, err := h.userStore.GetUserByID(ledger.LenderUserID.String())
				if err != nil {
					utils.WriteError(c, http.StatusInternalServerError, err)
					return
				}
				payerUsernames = append(payerUsernames, user.Username)
				inserted[ledger.ID.String()] = nil
			}
		}

		// get ledger detail
		res := types.ExpenseResponseBrief{
			ExpenseID:      expense.ID,
			Description:    expense.Description,
			Total:          expense.Total,
			PayerUserIDs:   payerUserIDs,
			PayerUsernames: payerUsernames,
		}
		response = append(response, res)
	}

	utils.WriteJSON(c, http.StatusOK, response)
}

func (h *Handler) handleGetExpenseDetail(c *gin.Context) {
	// get expense id from param
	// check expense id exist and get group id
	expenseID := c.Param("expenseId")
	expense, err := h.store.GetExpenseByID(expenseID)
	if err == types.ErrExpenseNotExist {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	} else if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	groupID := expense.GroupID.String()

	// extract userid from jwt, check userid is permitted for the group
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	_, err = h.groupStore.GetGroupByIDAndUser(groupID, userID)
	if err == types.ErrPermissionDenied {
		utils.WriteError(c, http.StatusForbidden, err)
		return
	} else if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// get expense
	user, err := h.userStore.GetUserByID(expense.CreateByUserID.String())
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	username := user.Username
	items, err := h.store.GetItemsByExpenseID(expenseID)
	var itemRsp []types.ItemResponse
	for _, it := range items {
		item := types.ItemResponse{
			ItemID:       it.ID,
			ItemName:     it.Name,
			ItemSubTotal: it.Amount.Mul(it.UnitPrice),
		}
		itemRsp = append(itemRsp, item)
	}
	response := types.ExpenseResponse{
		ID:                expense.ID,
		Description:       expense.Description,
		CreatedByUserID:   expense.CreateByUserID,
		CreatedByUsername: username,
		ExpenseTypeId:     expense.ExpenseTypeID,
		SubTotal:          expense.SubTotal,
		TaxFeeTip:         expense.TaxFeeTip,
		Total:             expense.Total,
		Currency:          expense.Currency,
		Items:             itemRsp,
	}

	utils.WriteJSON(c, http.StatusOK, response)
}

func (h *Handler) handleUpdateExpense(c *gin.Context) {
	// get expense id from param
	// check expense id exist and get group id
	expenseID := c.Param("expenseId")
	expense, err := h.store.GetExpenseByID(expenseID)
	if err == types.ErrExpenseNotExist {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	} else if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// get expense update paylaod from body
	var payload types.ExpenseUpdatePayload
	if err := utils.ParseJSON(c, &payload); err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}

	// extract userid from jwt, check userid is permitted for the group
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	_, err = h.groupStore.GetGroupByIDAndUser(payload.GroupID.String(), userID)
	if err == types.ErrPermissionDenied {
		utils.WriteError(c, http.StatusForbidden, err)
		return
	} else if err == types.ErrGroupNotExist {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	} else if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// update items
	items := payload.Items
	for _, it := range items {
		item := types.Item{
			ID:        it.ID,
			ExpenseID: expense.ID,
			Name:      it.ItemName,
			Amount:    it.Amount,
			Unit:      it.Unit,
			UnitPrice: it.UnitPrice,
		}
		if it.ID == uuid.Nil {
			item.ID = uuid.New()

			err := h.store.CreateItem(item)
			if err != nil {
				utils.WriteError(c, http.StatusInternalServerError, err)
				return
			}
		} else {
			err := h.store.UpdateItem(item)
			if err != nil {
				utils.WriteError(c, http.StatusInternalServerError, err)
				return
			}
		}
	}

	// update ledgers
	ledgers := payload.Ledgers
	for _, led := range ledgers {
		lenderID, err := uuid.Parse(led.LenderUserID)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		borrowerID, err := uuid.Parse(led.BorrowerUesrID)
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		ledger := types.Ledger{
			ID:             led.ID,
			ExpenseID:      expense.ID,
			LenderUserID:   lenderID,
			BorrowerUesrID: borrowerID,
			Share:          led.Share,
		}

		if led.ID == uuid.Nil {
			ledger.ID = uuid.New()

			err := h.store.CreateLedger(ledger)
			if err != nil {
				utils.WriteError(c, http.StatusInternalServerError, err)
				return
			}
		} else {
			err := h.store.UpdateLedger(ledger)
			if err != nil {
				utils.WriteError(c, http.StatusInternalServerError, err)
				return
			}
		}

	}

	// update expense
	id := expense.ID
	expense = &types.Expense{
		ID:             id,
		Description:    payload.Description,
		GroupID:        payload.GroupID,
		CreateByUserID: payload.CreateByUserID,
		ExpenseTypeID:  payload.ExpenseTypeID,
		ProviderName:   payload.ProviderName,
		SubTotal:       payload.SubTotal,
		TaxFeeTip:      payload.TaxFeeTip,
		Total:          payload.Total,
		Currency:       payload.Currency,
		InvoicePicUrl:  payload.InvoicePicUrl,
	}
	err = h.store.UpdateExpense(*expense)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}

func (h *Handler) handleSettleExpense(c *gin.Context) {
	// get group id from param
	groupID := c.Param("groupId")

	// get user id from jwt claim
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	// check user have permission for the group
	_, err = h.groupStore.GetGroupByIDAndUser(groupID, userID)
	if err == types.ErrPermissionDenied {
		utils.WriteError(c, http.StatusForbidden, err)
		return
	} else if err == types.ErrGroupNotExist {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	} else if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// settle group
	err = h.store.UpdateExpenseSettleInGroup(groupID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusCreated, nil)
}

func (h *Handler) handleGetUnsettledBalance(c *gin.Context) {
	// get group id from param
	groupID := c.Param("groupId")

	// get user id from jwt claim
	userID, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	// check user have permission for the group
	_, err = h.groupStore.GetGroupByIDAndUser(groupID, userID)
	if err == types.ErrPermissionDenied {
		utils.WriteError(c, http.StatusForbidden, err)
		return
	} else if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	// get ledgers
	ledgers, err := h.store.GetLedgerUnsettledFromGroup(groupID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	balanceSimplified := DebtSimplify(ledgers)

	// make response
	groupCurrency, err := h.groupStore.GetGroupCurrency(groupID)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	var balances []types.BalanceRsp
	for _, balance := range balanceSimplified {
		senderUsername, err := h.userStore.GetUsernameByID(balance.SenderUserID.String())
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		receiverUsername, err := h.userStore.GetUsernameByID(balance.ReceiverUserID.String())
		if err != nil {
			utils.WriteError(c, http.StatusInternalServerError, err)
			return
		}
		res := types.BalanceRsp{
			SenderUserID:     balance.SenderUserID,
			SenderUesrname:   senderUsername,
			ReceiverUserID:   balance.ReceiverUserID,
			ReceiverUsername: receiverUsername,
			Balance:          balance.Share,
		}
		balances = append(balances, res)
	}

	response := types.BalanceResponse{
		Currency: groupCurrency,
		Balances: balances,
	}

	utils.WriteJSON(c, http.StatusOK, response)
}
