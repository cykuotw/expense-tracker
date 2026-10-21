package expense

import (
	"context"
	"net/http/httptest"
	"testing"

	"expense-tracker/backend/types"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

type receiptViewTestStore struct {
	*mockExpenseStore
	receipt *types.ExpenseReceipt
}

func (s *receiptViewTestStore) GetExpenseReceiptByRole(uuid.UUID, string) (*types.ExpenseReceipt, error) {
	return s.receipt, nil
}

func TestReceiptViewSignsOnlyFinalizedVisibleReceiptForActiveAccount(t *testing.T) {
	for _, test := range []struct {
		name            string
		active, deleted bool
		state           string
		status          int
	}{
		{"finalized", true, false, "finalized", 200},
		{"inactive", false, false, "finalized", 403},
		{"soft deleted", true, true, "finalized", 404},
		{"pending", true, false, "pending", 404},
		{"deleting", true, false, "deleting", 404},
	} {
		t.Run(test.name, func(t *testing.T) {
			store := &receiptViewTestStore{mockExpenseStore: &mockExpenseStore{}, receipt: &types.ExpenseReceipt{Status: test.state, RetainedKey: "retained/private-key.jpg"}}
			users := userStoreMock()
			users.GetUserByIDFn = func(string) (*types.User, error) { return &types.User{IsActive: test.active}, nil }
			h := &Handler{store: store, userStore: users}
			calls := 0
			h.receiptView = func(_ context.Context, key string) (string, error) {
				calls++
				require.Equal(t, "retained/private-key.jpg", key)
				return "https://receipt.example/signed", nil
			}
			router := gin.New()
			router.POST("/view", func(c *gin.Context) {
				c.Set("userID", uuid.NewString())
				c.Set("expense", &types.Expense{ID: uuid.New(), IsDeleted: test.deleted})
				h.handleReceiptView(c)
			})
			response := httptest.NewRecorder()
			router.ServeHTTP(response, httptest.NewRequest("POST", "/view", nil))
			require.Equal(t, test.status, response.Code, response.Body.String())
			require.Contains(t, response.Header().Get("Cache-Control"), "no-store")
			if test.status == 200 {
				require.Equal(t, 1, calls)
				require.Contains(t, response.Body.String(), "expiresAt")
				require.NotContains(t, response.Body.String(), "retained/private-key")
			} else {
				require.Zero(t, calls)
			}
		})
	}
}
