package expense

import (
	"errors"
	"log/slog"

	"expense-tracker/backend/internal/observability"
	"expense-tracker/backend/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const settlementEvent = "expense_settlement"
const settlementLoggedKey = "expense_settlement_logged"

func logSettlementOutcome(c *gin.Context, operation string, actorID uuid.UUID, groupID, outcome, stage string) {
	c.Set(settlementLoggedKey, true)
	actor := "unavailable"
	if actorID != uuid.Nil {
		actor = actorID.String()
	}
	group := "unavailable"
	if parsed, err := uuid.Parse(groupID); err == nil {
		group = parsed.String()
	}

	observability.Logger(c).Info(settlementEvent,
		slog.String("actor_id", actor),
		slog.String("group_id", group),
		slog.String("operation", operation),
		slog.String("outcome", outcome),
		slog.String("stage", stage),
	)
}

func (h *Handler) observeSettlementGuard(operation string) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()
		if logged, _ := c.Get(settlementLoggedKey); logged == true {
			return
		}

		actorID, _ := uuid.Parse(c.GetString("userID"))
		outcome := "rejected"
		if c.Writer.Status() >= 500 {
			outcome = "failed"
		}
		logSettlementOutcome(c, operation, actorID, c.Param("groupId"), outcome, "route_guard")
	}
}

func settlementErrorOutcome(err error) string {
	switch {
	case errors.Is(err, types.ErrGroupNotExist),
		errors.Is(err, types.ErrGroupParticipantNotAllowed),
		errors.Is(err, types.ErrBalanceNotExist),
		errors.Is(err, types.ErrUserNotPermitted):
		return "rejected"
	default:
		return "failed"
	}
}

func settlementErrorStage(err error, fallback string) string {
	var rebuildError *balanceRebuildError
	if errors.As(err, &rebuildError) {
		return rebuildError.stage
	}
	return fallback
}
