package expense

import (
	"bytes"
	"encoding/json"
	"expense-tracker/config"
	"expense-tracker/services/auth"
	"expense-tracker/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestRouteUpdateExpenseDetail(t *testing.T) {
	store := &mockUpdateExpenseDetailStore{}
	userStore := &mockUpdateExpenseDetailUserStore{}
	groupStore := &mockUpdateExpenseDetailGroupStore{}
	controller := &mockExpenseController{}

	handler := NewHandler(store, userStore, groupStore, controller)

	type testcase struct {
		name             string
		payload          types.ExpenseUpdatePayload
		expenseID        string
		expectFail       bool
		expectStatusCode int
	}

	subtests := []testcase{
		{
			name: "valid",
			payload: types.ExpenseUpdatePayload{
				GroupID:       mockGroupID,
				PayByUserId:   mockCreatorID.String(),
				ExpenseTypeID: mockExpenseTypeID,
			},
			expenseID:        mockExpenseID.String(),
			expectFail:       false,
			expectStatusCode: http.StatusCreated,
		},
		{
			name: "invalid expense id",
			payload: types.ExpenseUpdatePayload{
				GroupID:       mockGroupID,
				PayByUserId:   mockCreatorID.String(),
				ExpenseTypeID: mockExpenseTypeID,
			},
			expenseID:        uuid.NewString(),
			expectFail:       true,
			expectStatusCode: http.StatusBadRequest,
		},
		{
			name: "invalid group id",
			payload: types.ExpenseUpdatePayload{
				GroupID:       uuid.New(),
				PayByUserId:   mockCreatorID.String(),
				ExpenseTypeID: mockExpenseTypeID,
			},
			expenseID:        mockExpenseID.String(),
			expectFail:       true,
			expectStatusCode: http.StatusForbidden,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			marshalled, _ := json.Marshal(test.payload)
			req, err := http.NewRequest(http.MethodPut, "/expense/"+test.expenseID, bytes.NewBuffer(marshalled))
			if err != nil {
				t.Fatal()
			}

			jwt, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), mockUserID)
			if err != nil {
				t.Fatal(err)
			}
			req.Header = map[string][]string{
				"Authorization": {"Bearer " + jwt},
			}

			rr := httptest.NewRecorder()
			gin.SetMode(gin.ReleaseMode)
			router := gin.New()
			router.PUT("/expense/:expenseId", handler.handleUpdateExpense)

			router.ServeHTTP(rr, req)

			assert.Equal(t, test.expectStatusCode, rr.Code)
		})
	}
}

type mockUpdateExpenseDetailStore struct {
	mockExpenseStore
}

func (s *mockUpdateExpenseDetailStore) GetExpenseByID(expenseID string) (*types.Expense, error) {
	if expenseID != mockExpenseID.String() {
		return nil, types.ErrExpenseNotExist
	}
	expense := &types.Expense{
		ID:      mockExpenseID,
		GroupID: mockGroupID,
	}
	return expense, nil
}
func (m *mockUpdateExpenseDetailStore) CheckExpenseExistByID(id string) (bool, error) {
	if id == mockExpenseID.String() {
		return true, nil
	}
	return false, nil
}

type mockUpdateExpenseDetailGroupStore struct {
	mockGroupStore
}

func (s *mockUpdateExpenseDetailGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	if groupID != mockGroupID.String() {
		return nil, types.ErrGroupNotExist
	}
	if userID != mockUserID.String() {
		return nil, types.ErrUserNotPermitted
	}
	return nil, nil
}
func (m *mockUpdateExpenseDetailGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	if groupId == mockGroupID.String() && userId == mockUserID.String() {
		return true, nil
	}
	return false, nil
}

type mockUpdateExpenseDetailUserStore struct {
	mockUserStore
}

func (m *mockUpdateExpenseDetailUserStore) GetUserByID(id string) (*types.User, error) {
	user := &types.User{
		ID:       mockUserID,
		Username: "test user",
	}
	return user, nil
}
