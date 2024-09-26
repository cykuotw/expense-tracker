package group

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

func TestGetGroupList(t *testing.T) {
	store := &mockGetGroupListStore{}
	userStore := &mockGetGroupListUserStore{}
	handler := NewHandler(store, userStore)

	t.Run("valid", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/groups", nil)
		if err != nil {
			t.Fatal(err)
		}

		jwt, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), mockUserId)
		if err != nil {
			t.Fatal(err)
		}
		req.Header = map[string][]string{
			"Authorization": {"Bearer " + jwt},
		}

		rr := httptest.NewRecorder()
		gin.SetMode(gin.ReleaseMode)
		router := gin.New()
		router.GET("/groups", handler.handleGetGroupList)

		router.ServeHTTP(rr, req)

		var rsp []types.GetGroupListResponse
		err = json.NewDecoder(rr.Body).Decode(&rsp)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, mockGroupListLen, len(rsp))
	})
	t.Run("invalid user", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, "/groups", nil)
		if err != nil {
			t.Fatal(err)
		}

		jwt, err := auth.CreateJWT([]byte(config.Envs.JWTSecret), uuid.New())
		if err != nil {
			t.Fatal(err)
		}
		req.Header = map[string][]string{
			"Authorization": {"Bearer " + jwt},
		}

		rr := httptest.NewRecorder()
		gin.SetMode(gin.ReleaseMode)
		router := gin.New()
		router.GET("/groups", handler.handleGetGroupList)

		router.ServeHTTP(rr, req)

		var rsp []types.GetGroupListResponse
		err = json.NewDecoder(rr.Body).Decode(&rsp)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, 0, len(rsp))
	})
}

var mockGroupListLen = 5

type mockGetGroupListStore struct{}

func (m *mockGetGroupListStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockGetGroupListStore) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}
func (s *mockGetGroupListStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	return nil, nil
}
func (m *mockGetGroupListStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	if userid != mockUserId.String() {
		return nil, nil
	}
	var groups []*types.Group

	for i := 0; i < mockGroupListLen; i++ {
		group := types.Group{
			ID:        uuid.New(),
			GroupName: uuid.New().String(),
		}
		groups = append(groups, &group)
	}
	return groups, nil
}
func (m *mockGetGroupListStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockGetGroupListStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockGetGroupListStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockGetGroupListStore) GetGroupCurrency(groupID string) (string, error) {
	return "", nil
}
func (m *mockGetGroupListStore) GetRelatedUser(currentUser string) ([]*types.GroupMember, error) {
	return nil, nil
}

type mockGetGroupListUserStore struct{}

func (m *mockGetGroupListUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetGroupListUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetGroupListUserStore) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetGroupListUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockGetGroupListUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockGetGroupListUserStore) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
