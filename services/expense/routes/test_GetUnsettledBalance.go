package expense

import (
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

func TestRouteGetUnsettledBalance(t *testing.T) {
	store := &mockGetUnsettledBalanceStore{}
	userStore := &mockGetUnsettledBalanceUserStore{}
	groupStore := &mockGetUnsettledBalanceGroupStore{}
	controller := &mockGetUnsettledBalanceController{}

	handler := NewHandler(store, userStore, groupStore, controller)

	type testcase struct {
		name             string
		groupID          string
		expectFail       bool
		expectStatusCode int
		expectResponse   types.BalanceResponse
	}

	subtests := []testcase{
		{
			name:             "valid",
			groupID:          mockGroupID.String(),
			expectFail:       false,
			expectStatusCode: http.StatusOK,
			expectResponse: types.BalanceResponse{
				Currency: mockCurrency,
				Balances: []types.BalanceRsp{
					{
						SenderUserID:   mockSenderIDs[0],
						ReceiverUserID: mockReceiverIDs[0],
					},
					{
						SenderUserID:   mockSenderIDs[1],
						ReceiverUserID: mockReceiverIDs[1],
					},
					{
						SenderUserID:   mockSenderIDs[2],
						ReceiverUserID: mockReceiverIDs[2],
					},
				},
			},
		},
		{
			name:             "invalid group id",
			groupID:          uuid.NewString(),
			expectFail:       true,
			expectStatusCode: http.StatusForbidden,
			expectResponse:   types.BalanceResponse{},
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			req, err := http.NewRequest(http.MethodGet, "/balance/"+test.groupID, nil)
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
			router.GET("/balance/:groupId", handler.handleGetUnsettledBalance)

			router.ServeHTTP(rr, req)

			var rsp types.BalanceResponse
			err = json.NewDecoder(rr.Body).Decode(&rsp)
			if err != nil {
				t.Fatal()
			}

			assert.Equal(t, test.expectStatusCode, rr.Code)
			assert.Equal(t, test.expectResponse.Currency, rsp.Currency)
			if assert.Equal(t, len(test.expectResponse.Balances), len(rsp.Balances)) {
				for i, b := range rsp.Balances {
					assert.Equal(t, test.expectResponse.Balances[i].SenderUserID, b.SenderUserID)
					assert.Equal(t, test.expectResponse.Balances[i].ReceiverUserID, b.ReceiverUserID)
				}
			}
		})
	}
}

var mockUserIDs = []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
var mockLedgerIDs = []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
var mockSenderIDs = []uuid.UUID{mockUserID, mockUserID, mockUserID}
var mockReceiverIDs = []uuid.UUID{uuid.New(), uuid.New(), uuid.New()}
var mockCurrency = "CAD"
var mockLedger = []*types.Ledger{
	{
		ID: mockLedgerIDs[0],
	},
	{
		ID: mockLedgerIDs[1],
	},
	{
		ID: mockLedgerIDs[2],
	},
}
var mockBalance = []*types.Balance{
	{
		SenderUserID:   mockSenderIDs[0],
		ReceiverUserID: mockReceiverIDs[0],
	},
	{
		SenderUserID:   mockSenderIDs[1],
		ReceiverUserID: mockReceiverIDs[1],
	},
	{
		SenderUserID:   mockSenderIDs[2],
		ReceiverUserID: mockReceiverIDs[2],
	},
}

type mockGetUnsettledBalanceStore struct {
	mockExpenseStore
}

func (s *mockGetUnsettledBalanceStore) GetLedgerUnsettledFromGroup(groupID string) ([]*types.Ledger, error) {
	if groupID != mockGroupID.String() {
		return nil, nil
	}

	return mockLedger, nil
}

type mockGetUnsettledBalanceGroupStore struct {
	mockGroupStore
}

func (m *mockGetUnsettledBalanceGroupStore) CheckGroupUserPairExist(groupId string, userId string) (bool, error) {
	if groupId == mockGroupID.String() && userId == mockUserID.String() {
		return true, nil
	}
	return false, nil
}

type mockGetUnsettledBalanceUserStore struct {
	mockUserStore
}

type mockGetUnsettledBalanceController struct {
	mockExpenseController
}

func (c *mockGetUnsettledBalanceController) DebtSimplify(ledgers []*types.Ledger) []*types.Balance {
	return mockBalance
}
