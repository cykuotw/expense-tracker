package expense

import (
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) handleSettleBalance(c *gin.Context) {
	groupId := c.Param("groupId")
	balanceId := c.Param("balanceId")
	actorID, err := uuid.Parse(c.GetString("userID"))
	if err != nil {
		logSettlementOutcome(c, "balance", uuid.Nil, groupId, "rejected", "actor_identity")
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidToken)
		return
	}

	stage := "transaction_begin"
	callbackCompleted := false
	err = h.store.RunInTransaction(func(store types.ExpenseTransactionStore) error {
		stage = "group_lock"
		if _, err := store.LockGroupCurrency(groupId); err != nil {
			return err
		}
		stage = "membership"
		if err := store.CheckGroupParticipants(groupId, []uuid.UUID{actorID}); err != nil {
			return err
		}
		stage = "balance_settle"
		if err := store.SettleBalanceByBalanceID(groupId, balanceId, actorID); err != nil {
			return err
		}
		stage = "completion_check"
		allSettled, err := store.CheckGroupBalanceAllSettled(groupId)
		if err != nil {
			return err
		}
		if !allSettled {
			callbackCompleted = true
			return nil
		}
		stage = "expense_settle"
		if err := store.UpdateExpenseSettleInGroup(groupId); err != nil {
			return err
		}
		callbackCompleted = true
		return nil
	})
	if err != nil {
		if callbackCompleted {
			stage = "transaction_commit"
		}
		logSettlementOutcome(c, "balance", actorID, groupId, settlementErrorOutcome(err), settlementErrorStage(err, stage))
		writeSettlementMutationError(c, err)
		return
	}

	logSettlementOutcome(c, "balance", actorID, groupId, "committed", "complete")
	utils.WriteJSON(c, http.StatusCreated, nil)
}
