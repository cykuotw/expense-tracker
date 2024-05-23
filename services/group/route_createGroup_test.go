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

func TestCreateGroup(t *testing.T) {
	store := &mockCreateGroupStore{}
	userStore := &mockCreateGroupUserStore{}
	handler := NewHandler(store, userStore)

	t.Run("valid", func(t *testing.T) {
		payload := types.CreateGroupPayload{
			GroupName:   "testgroup",
			Description: "testdesc",
		}
		marshalled, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, "/create_group", bytes.NewBuffer(marshalled))
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
		router.POST("/create_group", handler.handleCreateGroup)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
	})
	t.Run("valid-empty group name", func(t *testing.T) {
		payload := types.CreateGroupPayload{
			GroupName:   "",
			Description: "testdesc",
		}
		marshalled, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPost, "/create_group", bytes.NewBuffer(marshalled))
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
		router.POST("/create_group", handler.handleCreateGroup)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
	})
}

var mockUserId = uuid.New()

type mockCreateGroupStore struct{}

func (m *mockCreateGroupStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockCreateGroupStore) GetGroupByID(id string) (*types.Group, error) {
	return nil, nil
}
func (s *mockCreateGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	return nil, nil
}
func (m *mockCreateGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockCreateGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockCreateGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockCreateGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}

type mockCreateGroupUserStore struct{}

func (m *mockCreateGroupUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockCreateGroupUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockCreateGroupUserStore) GetUserByID(id string) (*types.User, error) {
	user := types.User{
		ID: mockUserId,
	}
	return &user, nil
}
func (m *mockCreateGroupUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockCreateGroupUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
