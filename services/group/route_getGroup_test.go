package group

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

func TestGetGroup(t *testing.T) {
	store := &mockGetGroupStore{}
	userStore := &mockGetGroupUserStore{}
	handler := NewHandler(store, userStore)

	t.Run("valid", func(t *testing.T) {
		payload := types.GetGroupResponse{
			GroupName:   "testgroup",
			Description: "testdesc",
		}
		marshalled, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodGet, "/group/"+mockGroupId.String(),
			bytes.NewBuffer(marshalled))
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
		router.GET("/group/:groupid", handler.handleGetGroup)

		router.ServeHTTP(rr, req)

		var rsp types.GetGroupResponse
		err = json.NewDecoder(rr.Body).Decode(&rsp)

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, mockMemberNum, len(rsp.Members))
	})
	t.Run("invalid userid", func(t *testing.T) {
		payload := types.GetGroupResponse{
			GroupName:   "testgroup",
			Description: "testdesc",
		}
		marshalled, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodGet, "/group/"+mockGroupId.String(),
			bytes.NewBuffer(marshalled))
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
		router.GET("/group/:groupid", handler.handleGetGroup)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
	t.Run("invalid groupid", func(t *testing.T) {
		payload := types.GetGroupResponse{
			GroupName:   "testgroup",
			Description: "testdesc",
		}
		marshalled, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodGet, "/group/"+uuid.New().String(),
			bytes.NewBuffer(marshalled))
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
		router.GET("/group/:groupid", handler.handleGetGroup)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

var mockGroupId = uuid.New()
var mockMemberNum = 5

type mockGetGroupStore struct{}

func (m *mockGetGroupStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockGetGroupStore) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}
func (s *mockGetGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	if userID != mockUserId.String() {
		return nil, types.ErrUserNotExist
	}
	if groupID != mockGroupId.String() {
		return nil, types.ErrGroupNotExist
	}

	return &types.Group{}, nil
}
func (m *mockGetGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockGetGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	var users []*types.User
	for i := 0; i < mockMemberNum; i++ {
		user := types.User{
			ID:       uuid.New(),
			Username: uuid.New().String(),
		}
		users = append(users, &user)
	}
	return users, nil
}
func (m *mockGetGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockGetGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockGetGroupStore) GetGroupCurrency(groupID string) (string, error) {
	return "", nil
}

type mockGetGroupUserStore struct{}

func (m *mockGetGroupUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetGroupUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockGetGroupUserStore) GetUserByID(id string) (*types.User, error) {
	user := types.User{
		ID: mockUserId,
	}
	return &user, nil
}
func (m *mockGetGroupUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockGetGroupUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
