package expense

import (
	"errors"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) handleSettleExpense(c *gin.Context) {
	// get group id from param
	groupID := c.Param("groupId")
	actorID, err := uuid.Parse(c.GetString("userID"))
	if err != nil {
		logSettlementOutcome(c, "group", uuid.Nil, groupID, "rejected", "actor_identity")
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidToken)
		return
	}

	// settle group
	stage := "transaction_begin"
	callbackCompleted := false
	err = h.store.RunInTransaction(func(store types.ExpenseTransactionStore) error {
		stage = "group_lock"
		if _, err := store.LockGroupCurrency(groupID); err != nil {
			return err
		}
		stage = "membership"
		if err := store.CheckGroupParticipants(groupID, []uuid.UUID{actorID}); err != nil {
			return err
		}
		stage = "expense_settle"
		if err := store.UpdateExpenseSettleInGroup(groupID); err != nil {
			return err
		}
		stage = "balance_rebuild"
		if err := h.updateBalanceWithStore(store, groupID); err != nil {
			return err
		}
		callbackCompleted = true
		return nil
	})
	if err != nil {
		if callbackCompleted {
			stage = "transaction_commit"
		}
		logSettlementOutcome(c, "group", actorID, groupID, settlementErrorOutcome(err), settlementErrorStage(err, stage))
		writeSettlementMutationError(c, err)
		return
	}

	logSettlementOutcome(c, "group", actorID, groupID, "committed", "complete")
	utils.WriteJSON(c, http.StatusCreated, nil)
}

func writeSettlementMutationError(c *gin.Context, err error) {
	if writeBalanceLedgerConflict(c, err) {
		return
	}

	switch {
	case errors.Is(err, types.ErrGroupNotExist),
		errors.Is(err, types.ErrGroupParticipantNotAllowed),
		errors.Is(err, types.ErrBalanceNotExist):
		utils.WriteError(c, http.StatusNotFound, err)
	case errors.Is(err, types.ErrUserNotPermitted):
		utils.WriteError(c, http.StatusForbidden, err)
	default:
		utils.WriteError(c, http.StatusInternalServerError, err)
	}
}
