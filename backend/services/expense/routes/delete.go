package expense

import (
	"errors"
	"expense-tracker/backend/services/middleware/extractors"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (h *Handler) handleDeleteExpense(c *gin.Context) {
	// get expense id from param
	// check expense id exist and get group id
	expense, err := extractors.GetExpenseFromStore(c)
	if err != nil {
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	actorID, err := uuid.Parse(c.GetString("userID"))
	if err != nil {
		utils.WriteError(c, http.StatusUnauthorized, types.ErrInvalidToken)
		return
	}

	err = h.store.RunInTransaction(func(store types.ExpenseTransactionStore) error {
		if _, err := store.LockGroupCurrency(expense.GroupID.String()); err != nil {
			return err
		}
		if err := store.CheckGroupParticipants(expense.GroupID.String(), []uuid.UUID{actorID}); err != nil {
			return err
		}
		if err := store.DeleteExpense(*expense); err != nil {
			return err
		}
		return h.updateBalanceWithStore(store, expense.GroupID.String())
	})
	if err != nil {
		if writeBalanceLedgerConflict(c, err) {
			return
		}
		if errors.Is(err, types.ErrGroupNotExist) || errors.Is(err, types.ErrGroupParticipantNotAllowed) {
			utils.WriteError(c, http.StatusNotFound, err)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}

	utils.WriteJSON(c, http.StatusOK, nil)
}
