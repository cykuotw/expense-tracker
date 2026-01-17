package group

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
	"github.com/stretchr/testify/assert"
)

var mockRequesterId = uuid.New()

func TestUpdateGroupMemberRoute(t *testing.T) {
	store := updateGroupMemberStoreMock()
	userStore := updateGroupMemberUserStoreMock()
	handler := NewHandler(store, userStore)

	// // define test cases
	type testcase struct {
		name             string
		mockAction       string
		mockRequesterID  string
		mockGroupID      string
		mockUserID       string
		expectFail       bool
		expectReturnCode int
	}

	subtests := []testcase{
		{
			name:             "valid add",
			mockAction:       "add",
			mockRequesterID:  mockRequesterId.String(),
			mockGroupID:      mockGroupId.String(),
			mockUserID:       mockUserId.String(),
			expectFail:       false,
			expectReturnCode: http.StatusCreated,
		},
		{
			name:             "valid delete",
			mockAction:       "delete",
			mockRequesterID:  mockRequesterId.String(),
			mockGroupID:      mockGroupId.String(),
			mockUserID:       mockUserId.String(),
			expectFail:       false,
			expectReturnCode: http.StatusCreated,
		},
		{
			name:             "invalid action",
			mockAction:       "",
			mockRequesterID:  mockRequesterId.String(),
			mockGroupID:      mockGroupId.String(),
			mockUserID:       mockUserId.String(),
			expectFail:       true,
			expectReturnCode: http.StatusBadRequest,
		},
		{
			name:             "invalid requester",
			mockAction:       "add",
			mockRequesterID:  uuid.NewString(),
			mockGroupID:      mockGroupId.String(),
			mockUserID:       mockUserId.String(),
			expectFail:       true,
			expectReturnCode: http.StatusForbidden,
		},
		{
			name:             "invalid group",
			mockAction:       "add",
			mockRequesterID:  mockRequesterId.String(),
			mockGroupID:      uuid.NewString(),
			mockUserID:       mockUserId.String(),
			expectFail:       true,
			expectReturnCode: http.StatusBadRequest,
		},
		{
			name:             "invalid user",
			mockAction:       "delete",
			mockRequesterID:  mockRequesterId.URN(),
			mockGroupID:      mockGroupId.String(),
			mockUserID:       uuid.NewString(),
			expectFail:       true,
			expectReturnCode: http.StatusBadRequest,
		},
	}

	for _, test := range subtests {
		t.Run(test.name, func(t *testing.T) {
			payload := types.UpdateGroupMemberPayload{
				Action:  test.mockAction,
				GroupID: test.mockGroupID,
				UserID:  test.mockUserID,
			}

			marshalled, _ := json.Marshal(payload)
			req, err := http.NewRequest(http.MethodPut, "/group_member", bytes.NewBuffer(marshalled))
			if err != nil {
				t.Fatal(err)
			}
			jwt, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), uuid.MustParse(test.mockRequesterID))
			if err != nil {
				t.Fatal(err)
			}
			req.Header = map[string][]string{
				"Authorization": {"Bearer " + jwt},
			}

			rr := httptest.NewRecorder()
			gin.SetMode(gin.ReleaseMode)
			router := gin.New()
			router.PUT("/group_member", handler.handleUpdateGroupMember)

			router.ServeHTTP(rr, req)

			assert.Equal(t, test.expectReturnCode, rr.Code)
		})
	}

}
