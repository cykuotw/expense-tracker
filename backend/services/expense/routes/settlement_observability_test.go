package expense

import (
	"bytes"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"expense-tracker/backend/internal/observability"
	"expense-tracker/backend/types"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
)

func TestSettlementDiagnosticsAreCorrelatedAndSanitized(t *testing.T) {
	tests := []struct {
		name          string
		operation     string
		configure     func(*mockExpenseStore)
		wantStatus    int
		wantOutcome   string
		wantStage     string
		forbiddenText string
	}{
		{
			name:        "group settlement committed",
			operation:   "group",
			wantStatus:  http.StatusCreated,
			wantOutcome: "committed",
			wantStage:   "complete",
		},
		{
			name:      "balance party rejected",
			operation: "balance",
			configure: func(store *mockExpenseStore) {
				store.SettleBalanceByBalanceIDFn = func(string, string, uuid.UUID) error {
					return types.ErrUserNotPermitted
				}
			},
			wantStatus:  http.StatusForbidden,
			wantOutcome: "rejected",
			wantStage:   "balance_settle",
		},
		{
			name:      "balance rebuild failed without logging the database error",
			operation: "group",
			configure: func(store *mockExpenseStore) {
				store.GetLedgerUnsettledFromGroupFn = func(string) ([]*types.Ledger, error) {
					return nil, errors.New("database-secret-must-not-be-logged")
				}
			},
			wantStatus:    http.StatusInternalServerError,
			wantOutcome:   "failed",
			wantStage:     "ledger_read",
			forbiddenText: "database-secret-must-not-be-logged",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := settleExpenseStoreMock()
			if test.configure != nil {
				test.configure(store)
			}

			response, output := runLoggedSettlementHandler(t, store, test.operation)

			require.Equal(t, test.wantStatus, response.Code)
			event := findLogEvent(t, output, settlementEvent)
			require.Equal(t, "settlement-request-id", event["request_id"])
			require.Equal(t, mockUserID.String(), event["actor_id"])
			require.Equal(t, mockGroupID.String(), event["group_id"])
			require.Equal(t, test.operation, event["operation"])
			require.Equal(t, test.wantOutcome, event["outcome"])
			require.Equal(t, test.wantStage, event["stage"])
			if test.forbiddenText != "" {
				require.NotContains(t, output, test.forbiddenText)
			}
		})
	}
}

func TestSettlementGuardRejectionIsDiagnosed(t *testing.T) {
	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	handler := NewHandler(nil, nil, nil, nil)
	router := gin.New()
	router.Use(observability.RequestLogging(observability.RequestLoggingConfig{
		Logger:       logger,
		NewRequestID: func() string { return "guard-request-id" },
	}))
	router.PUT("/settle_expense/:groupId",
		handler.observeSettlementGuard("group"),
		func(c *gin.Context) {
			c.Set("userID", mockUserID.String())
			c.AbortWithStatus(http.StatusNotFound)
		},
	)

	response := httptest.NewRecorder()
	router.ServeHTTP(response, httptest.NewRequest(http.MethodPut, "/settle_expense/"+mockGroupID.String(), nil))

	require.Equal(t, http.StatusNotFound, response.Code)
	event := findLogEvent(t, output.String(), settlementEvent)
	require.Equal(t, "guard-request-id", event["request_id"])
	require.Equal(t, mockUserID.String(), event["actor_id"])
	require.Equal(t, "rejected", event["outcome"])
	require.Equal(t, "route_guard", event["stage"])
}

func runLoggedSettlementHandler(t *testing.T, store *mockExpenseStore, operation string) (*httptest.ResponseRecorder, string) {
	t.Helper()

	gin.SetMode(gin.TestMode)
	var output bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&output, nil))
	router := gin.New()
	router.Use(observability.RequestLogging(observability.RequestLoggingConfig{
		Logger:       logger,
		NewRequestID: func() string { return "settlement-request-id" },
	}))
	handler := NewHandler(store, nil, nil, expenseControllerMock())
	if operation == "balance" {
		router.POST("/settle_balance/:groupId/:balanceId", func(c *gin.Context) {
			c.Set("userID", mockUserID.String())
			handler.handleSettleBalance(c)
		})
	} else {
		router.PUT("/settle_expense/:groupId", func(c *gin.Context) {
			c.Set("userID", mockUserID.String())
			handler.handleSettleExpense(c)
		})
	}

	path := "/settle_expense/" + mockGroupID.String()
	method := http.MethodPut
	if operation == "balance" {
		path = "/settle_balance/" + mockGroupID.String() + "/" + uuid.NewString()
		method = http.MethodPost
	}
	request := httptest.NewRequest(method, path, nil)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	return response, output.String()
}

func findLogEvent(t *testing.T, output, eventName string) map[string]any {
	t.Helper()

	for line := range strings.SplitSeq(strings.TrimSpace(output), "\n") {
		var event map[string]any
		require.NoError(t, json.Unmarshal([]byte(line), &event))
		if event["msg"] == eventName {
			return event
		}
	}
	t.Fatalf("log event %q not found in %s", eventName, output)
	return nil
}
