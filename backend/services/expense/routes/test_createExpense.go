package expense

import (
	"bytes"
	"encoding/json"
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
)

func TestRouteCreateExpense(t *testing.T) {
	store := &mockCreateExpenseStore{}
	userStore := &mockCreateExpenseUserStore{}
	groupStore := &mockCreateExpenseGroupStore{}
	controller := &mockExpenseController{}

	handler := NewHandler(store, userStore, groupStore, controller)

	type testcase struct {
		name             string
		payload          types.ExpensePayload
		expectFail       bool
		expectStatusCode int
	}

	subtests := []testcase{
		{
			name: "valid",
			payload: types.ExpensePayload{
				Description:    "test desc",
				GroupID:        mockGroupID.String(),
				CreateByUserID: mockCreatorID.String(),
				PayByUserId:    mockPayerID.String(),
				ExpenseTypeID:  mockExpenseTypeID.String(),
				ProviderName:   "test provider",
				SubTotal:       decimal.NewFromFloat(20.1),
				TaxFeeTip:      decimal.NewFromFloat(2.1),
				Total:          decimal.NewFromFloat(22.2),
				Currency:       "CAD",
				Items:          nil,
				Ledgers:        nil,
			},
			expectFail:       false,
			expectStatusCode: http.StatusCreated,
		},
		{
			name: "invalid user id",
			payload: types.ExpensePayload{
				Description:    "test desc",
				GroupID:        mockGroupID.String(),
				CreateByUserID: uuid.NewString(),
				PayByUserId:    mockPayerID.String(),
				ExpenseTypeID:  mockExpenseTypeID.String(),
				ProviderName:   "test provider",
				SubTotal:       decimal.NewFromFloat(20.1),
				TaxFeeTip:      decimal.NewFromFloat(2.1),
				Total:          decimal.NewFromFloat(22.2),
				Currency:       "CAD",
				Items:          nil,
				Ledgers:        nil,
			},
			expectFail:       true,
			expectStatusCode: http.StatusBadRequest,
		},
		{
			name: "invalid group id",
			payload: types.ExpensePayload{
				Description:    "test desc",
				GroupID:        uuid.NewString(),
				CreateByUserID: mockCreatorID.String(),
				PayByUserId:    mockPayerID.String(),
				ExpenseTypeID:  mockExpenseTypeID.String(),
				ProviderName:   "test provider",
				SubTotal:       decimal.NewFromFloat(20.1),
				TaxFeeTip:      decimal.NewFromFloat(2.1),
				Total:          decimal.NewFromFloat(22.2),
				Currency:       "CAD",
				Items:          nil,
				Ledgers:        nil,
			},
			expectFail:       true,
			expectStatusCode: http.StatusBadRequest,
		},
		{
			name: "invalid group id",
			payload: types.ExpensePayload{
				Description:    "test desc",
				GroupID:        mockGroupID.String(),
				CreateByUserID: mockUserID.String(),
				PayByUserId:    mockPayerID.String(),
				ExpenseTypeID:  mockExpenseTypeID.String(),
				ProviderName:   "test provider",
				SubTotal:       decimal.NewFromFloat(20.1),
				TaxFeeTip:      decimal.NewFromFloat(2.1),
				Total:          decimal.NewFromFloat(22.2),
				Currency:       "CAD",
				Items:          nil,
				Ledgers:        nil,
			},
			expectFail:       true,
			expectStatusCode: http.StatusForbidden,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			marshalled, _ := json.Marshal(test.payload)
			req, err := http.NewRequest(http.MethodPost, "/create_expense", bytes.NewBuffer(marshalled))
			if err != nil {
				t.Fatal(err)
			}

			jwt, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), uuid.MustParse(test.payload.CreateByUserID))
			if err != nil {
				t.Fatal(err)
			}
			req.Header = map[string][]string{
				"Authorization": {"Bearer " + jwt},
			}

			rr := httptest.NewRecorder()
			gin.SetMode(gin.ReleaseMode)
			router := gin.New()
			router.POST("/create_expense", handler.handleCreateExpense)

			router.ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatusCode, rr.Code)
		})
	}
}

type mockCreateExpenseStore struct {
	mockExpenseStore
}

type mockCreateExpenseGroupStore struct {
	mockGroupStore
}

func (m *mockCreateExpenseGroupStore) CheckGroupExistById(id string) (bool, error) {
	if id == mockGroupID.String() {
		return true, nil
	}
	return false, nil
}
func (m *mockCreateExpenseGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	if (groupId == mockGroupID.String()) && (userId == mockCreatorID.String()) {
		return true, nil
	}
	return false, nil
}

type mockCreateExpenseUserStore struct {
	mockUserStore
}

func (m *mockCreateExpenseUserStore) CheckUserExistByID(id string) (bool, error) {
	if id == mockCreatorID.String() || id == mockUserID.String() {
		return true, nil
	}
	return false, nil
}

type mockExpenseController struct {
	mockController
}
