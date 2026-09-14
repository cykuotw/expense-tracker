package monthlyreview

import (
	"context"
	"errors"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"expense-tracker/backend/utils"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReviewStore interface {
	Get(context.Context, uuid.UUID, uuid.UUID, time.Time, time.Time) (Review, error)
	GetTrend(context.Context, uuid.UUID, uuid.UUID, time.Time, time.Time) (Trend, error)
	ListExpenses(context.Context, uuid.UUID, uuid.UUID, time.Time, string, string) (ExpensePage, error)
}

type Handler struct {
	store ReviewStore
	now   func() time.Time
}

func NewHandler(store ReviewStore) *Handler {
	return &Handler{store: store, now: time.Now}
}

func (h *Handler) RegisterRoutes(router *gin.RouterGroup) {
	router.GET("/group/:groupid/monthly-review/:month", h.get)
	router.GET("/group/:groupid/monthly-review/:month/expenses", h.getExpenses)
	router.GET("/group/:groupid/monthly-review-trend/:month", h.getTrend)
}

func (h *Handler) getExpenses(c *gin.Context) {
	groupID, userID, err := requestScope(c)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, types.ErrGroupNotExist) {
			status = http.StatusNotFound
		}
		utils.WriteError(c, status, err)
		return
	}
	month, err := ParseMonth(c.Param("month"))
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	currency := c.Query("currency")
	if currency == "" || len(currency) > 16 {
		utils.WriteError(c, http.StatusBadRequest, errors.New("currency is required"))
		return
	}
	page, err := h.store.ListExpenses(c.Request.Context(), groupID, userID, month, currency, c.Query("cursor"))
	if err != nil {
		switch {
		case errors.Is(err, types.ErrGroupNotExist):
			utils.WriteError(c, http.StatusNotFound, types.ErrGroupNotExist)
		case errors.Is(err, ErrInvalidExpenseCursor):
			utils.WriteError(c, http.StatusBadRequest, err)
		default:
			utils.WriteError(c, http.StatusInternalServerError, err)
		}
		return
	}
	utils.WriteJSON(c, http.StatusOK, page)
}

func requestScope(c *gin.Context) (uuid.UUID, uuid.UUID, error) {
	groupID, err := uuid.Parse(c.Param("groupid"))
	if err != nil {
		return uuid.Nil, uuid.Nil, types.ErrGroupNotExist
	}
	userValue, err := auth.ExtractJWTClaim(c, "userID")
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	userID, err := uuid.Parse(userValue)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	return groupID, userID, nil
}

func (h *Handler) get(c *gin.Context) {
	groupID, userID, err := requestScope(c)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, types.ErrGroupNotExist) {
			status = http.StatusNotFound
		}
		utils.WriteError(c, status, err)
		return
	}
	month, err := ParseMonth(c.Param("month"))
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	review, err := h.store.Get(c.Request.Context(), groupID, userID, month, h.now().UTC())
	if err != nil {
		if errors.Is(err, types.ErrGroupNotExist) {
			utils.WriteError(c, http.StatusNotFound, types.ErrGroupNotExist)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, review)
}

func (h *Handler) getTrend(c *gin.Context) {
	groupID, userID, err := requestScope(c)
	if err != nil {
		status := http.StatusUnauthorized
		if errors.Is(err, types.ErrGroupNotExist) {
			status = http.StatusNotFound
		}
		utils.WriteError(c, status, err)
		return
	}
	month, err := ParseMonth(c.Param("month"))
	if err != nil {
		utils.WriteError(c, http.StatusBadRequest, err)
		return
	}
	trend, err := h.store.GetTrend(c.Request.Context(), groupID, userID, month, h.now().UTC())
	if err != nil {
		if errors.Is(err, types.ErrGroupNotExist) {
			utils.WriteError(c, http.StatusNotFound, types.ErrGroupNotExist)
			return
		}
		utils.WriteError(c, http.StatusInternalServerError, err)
		return
	}
	utils.WriteJSON(c, http.StatusOK, trend)
}
