package group

import (
	"encoding/json"
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"expense-tracker/backend/types"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/require"
)

type lifecycleGroupMock struct {
	*mockGroupStore
	list func(string, string, int) (types.ArchivedGroupPage, error)
}

func (m lifecycleGroupMock) GetGroupLifecycle(string, string) (types.GroupLifecycle, error) {
	return types.GroupLifecycle{IsActive: false, CanRestore: true, CanManageLifecycle: true}, nil
}
func (m lifecycleGroupMock) GetArchivedGroups(userID, cursor string, limit int) (types.ArchivedGroupPage, error) {
	return m.list(userID, cursor, limit)
}

func TestArchivedListRequestBoundsAndIdentity(t *testing.T) {
	for _, test := range []struct {
		query         string
		status, limit int
	}{
		{"", 200, 20}, {"?limit=100&cursor=next", 200, 100}, {"?limit=0", 400, 0},
		{"?limit=101", 400, 0}, {"?limit=invalid", 400, 0}, {"?cursor=invalid", 400, 20},
	} {
		t.Run(test.query, func(t *testing.T) {
			calls := 0
			store := lifecycleGroupMock{mockGroupStore: groupStoreMock(), list: func(actor, cursor string, limit int) (types.ArchivedGroupPage, error) {
				calls++
				require.Equal(t, mockUserId.String(), actor)
				require.Equal(t, test.limit, limit)
				if cursor == "invalid" {
					return types.ArchivedGroupPage{}, types.ErrInvalidGroupCursor
				}
				return types.ArchivedGroupPage{Groups: []types.GetGroupListResponse{}, NextCursor: "next"}, nil
			}}
			router := gin.New()
			router.GET("/groups/archived", NewHandler(store, userStoreMock()).handleGetArchivedGroups)
			token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), mockUserId)
			require.NoError(t, err)
			request := httptest.NewRequest(http.MethodGet, "/groups/archived"+test.query, nil)
			request.Header.Set("Authorization", "Bearer "+token)
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			require.Equal(t, test.status, response.Code)
			if test.limit == 0 {
				require.Zero(t, calls)
			}
			if test.status == 200 {
				var page types.ArchivedGroupPage
				require.NoError(t, json.Unmarshal(response.Body.Bytes(), &page))
				require.NotNil(t, page.Groups)
				require.Equal(t, "next", page.NextCursor)
			}
		})
	}
}

func TestLifecycleRouteUsesAuthenticatedCreatorAndMapsConflict(t *testing.T) {
	for _, active := range []bool{false, true} {
		store := archiveGroupStoreMock()
		calls := 0
		store.UpdateGroupStatusFn = func(groupID, creatorID string, desired bool) error {
			calls++
			require.Equal(t, mockGroupId.String(), groupID)
			require.Equal(t, mockUserId.String(), creatorID)
			require.Equal(t, active, desired)
			if !active {
				return types.ErrGroupUnsettled
			}
			return nil
		}
		h := NewHandler(store, userStoreMock())
		router := gin.New()
		path := "/archive_group/"
		handler := h.handleArchiveGroup
		if active {
			path = "/restore_group/"
			handler = h.handleRestoreGroup
		}
		router.PUT(path+":groupId", handler)
		token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), mockUserId)
		require.NoError(t, err)
		request := httptest.NewRequest(http.MethodPut, path+mockGroupId.String(), nil)
		request.Header.Set("Authorization", "Bearer "+token)
		response := httptest.NewRecorder()
		router.ServeHTTP(response, request)
		if active {
			require.Equal(t, http.StatusCreated, response.Code)
		} else {
			require.Equal(t, http.StatusConflict, response.Code)
		}
		require.Equal(t, 1, calls)
	}
}

func TestLifecycleRouteRejectsNonCreatorWithoutMutation(t *testing.T) {
	store := archiveGroupStoreMock()
	store.UpdateGroupStatusFn = func(string, string, bool) error { t.Fatal("unauthorized mutation"); return nil }
	h := NewHandler(store, userStoreMock())
	router := gin.New()
	router.PUT("/restore_group/:groupId", h.handleRestoreGroup)
	token, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), mockRequesterId)
	require.NoError(t, err)
	request := httptest.NewRequest(http.MethodPut, "/restore_group/"+mockGroupId.String(), nil)
	request.Header.Set("Authorization", "Bearer "+token)
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	require.Equal(t, http.StatusNotFound, response.Code)
}
