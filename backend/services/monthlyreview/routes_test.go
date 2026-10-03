package monthlyreview

import (
	"context"
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type expenseRequestStore struct {
	ReviewStore
	options ExpenseListOptions
	calls   int
	err     error
}

func (s *expenseRequestStore) ListExpenses(_ context.Context, _, _ uuid.UUID, _ time.Time, _, _ string, options ...ExpenseListOptions) (ExpensePage, error) {
	s.calls++
	s.options = options[0]
	return ExpensePage{Expenses: []Expense{}}, s.err
}

func TestExpenseRequestOptions(t *testing.T) {
	token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), uuid.New())
	require.NoError(t, err)
	for _, test := range []struct {
		name, category, sort string
		storeErr             error
		status               int
	}{
		{name: "default", status: http.StatusOK},
		{name: "oldest", category: "Food and Drink", sort: "date_asc", status: http.StatusOK},
		{name: "highest", category: "Food and Drink", sort: "amount_desc", status: http.StatusOK},
		{name: "lowest", category: "Food and Drink", sort: "amount_asc", status: http.StatusOK},
		{name: "invalid sort", sort: "total; DROP TABLE expense", status: http.StatusBadRequest},
		{name: "oversized category", category: strings.Repeat("x", 201), status: http.StatusBadRequest},
		{name: "mismatched cursor", storeErr: ErrInvalidExpenseCursor, status: http.StatusBadRequest},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &expenseRequestStore{err: test.storeErr}
			router := gin.New()
			NewHandler(store).RegisterRoutes(router.Group(""))
			query := url.Values{"currency": {"CAD"}, "category": {test.category}, "sort": {test.sort}}
			request := httptest.NewRequest(http.MethodGet, "/group/"+uuid.NewString()+"/monthly-review/2026-08/expenses?"+query.Encode(), nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, test.status, response.Code)
			if test.status == http.StatusOK {
				require.Equal(t, 1, store.calls)
				expectedSort := test.sort
				if expectedSort == "" {
					expectedSort = "date_desc"
				}
				require.Equal(t, ExpenseListOptions{Category: test.category, Sort: expectedSort}, store.options)
			} else if test.storeErr == nil {
				require.Zero(t, store.calls)
			}
		})
	}
}
