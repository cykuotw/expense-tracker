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

func TestUpdateGroupMember(t *testing.T) {
	store := &mockUpdateGroupMemberStore{}
	userStore := &mockCreateGroupUserStore{}
	handler := NewHandler(store, userStore)

	t.Run("valid", func(t *testing.T) {
		payload := types.UpdateGroupMemberPayload{
			Action:  "add",
			GroupID: mockGroupId.String(),
			UserID:  uuid.New().String(),
		}
		marshalled, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPut, "/group_member", bytes.NewBuffer(marshalled))
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
		router.PUT("/group_member", handler.handleUpdateGroupMember)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
	})
	t.Run("invalid action", func(t *testing.T) {
		payload := types.UpdateGroupMemberPayload{
			Action:  "invalid",
			GroupID: mockGroupId.String(),
			UserID:  uuid.New().String(),
		}
		marshalled, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPut, "/group_member", bytes.NewBuffer(marshalled))
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
		router.PUT("/group_member", handler.handleUpdateGroupMember)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
	t.Run("invalid group id", func(t *testing.T) {
		payload := types.UpdateGroupMemberPayload{
			Action:  "add",
			GroupID: uuid.NewString(),
			UserID:  uuid.NewString(),
		}
		marshalled, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPut, "/group_member", bytes.NewBuffer(marshalled))
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
		router.PUT("/group_member", handler.handleUpdateGroupMember)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
	t.Run("invalid requester", func(t *testing.T) {
		payload := types.UpdateGroupMemberPayload{
			Action:  "add",
			GroupID: mockGroupId.String(),
			UserID:  uuid.New().String(),
		}
		marshalled, _ := json.Marshal(payload)
		req, err := http.NewRequest(http.MethodPut, "/group_member", bytes.NewBuffer(marshalled))
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
		router.PUT("/group_member", handler.handleUpdateGroupMember)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusForbidden, rr.Code)
	})
}

type mockUpdateGroupMemberStore struct{}

func (m *mockUpdateGroupMemberStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockUpdateGroupMemberStore) GetGroupByID(id string) (*types.Group, error) {
	if id != mockGroupId.String() {
		return nil, types.ErrGroupNotExist
	}
	groupid, _ := uuid.Parse(id)
	group := types.Group{
		ID: groupid,
	}
	return &group, nil
}
func (s *mockUpdateGroupMemberStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	if groupID != mockGroupId.String() {
		return nil, types.ErrGroupNotExist
	}
	if userID != mockUserId.String() {
		return nil, types.ErrUserNotExist
	}
	return nil, nil
}
func (m *mockUpdateGroupMemberStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockUpdateGroupMemberStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockUpdateGroupMemberStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockUpdateGroupMemberStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}

type mockUpdateGroupMemberUserStore struct{}

func (m *mockUpdateGroupMemberUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockUpdateGroupMemberUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockUpdateGroupMemberUserStore) GetUserByID(id string) (*types.User, error) {
	userid, _ := uuid.Parse(id)
	user := types.User{
		ID: userid,
	}
	return &user, nil
}
func (m *mockUpdateGroupMemberUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockUpdateGroupMemberUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
