package expense

import (
	"errors"
	"net/http"

	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"

	"github.com/gin-gonic/gin"
)

func writeBalanceLedgerConflict(c *gin.Context, err error) bool {
	if !errors.Is(err, types.ErrBalanceLedgerConflict) {
		return false
	}

	utils.WriteError(c, http.StatusConflict, err)
	return true
}
