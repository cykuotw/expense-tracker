package group

import (
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

func TestArchiveGroup(t *testing.T) {
	store := &mockArchiveGroupStore{}
	userStore := &mockArchiveGroupUserStore{}
	handler := NewHandler(store, userStore)

	t.Run("valid", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPut, "/archive_group/"+mockGroupId.String(), nil)
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
		router.PUT("/archive_group/:groupId", handler.handleArchiveGroup)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusCreated, rr.Code)
	})
	t.Run("invalid group id", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodPut, "/archive_group/"+uuid.NewString(), nil)
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
		router.PUT("/archive_group/:groupId", handler.handleArchiveGroup)

		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
	})
}

type mockArchiveGroupStore struct{}

func (m *mockArchiveGroupStore) CreateGroup(group types.Group) error {
	return nil
}
func (m *mockArchiveGroupStore) GetGroupByID(id string) (*types.Group, error) {
	if id != mockGroupId.String() {
		return nil, types.ErrGroupNotExist
	}
	return nil, nil
}
func (s *mockArchiveGroupStore) GetGroupByIDAndUser(groupID string, userID string) (*types.Group, error) {
	return nil, nil
}
func (m *mockArchiveGroupStore) GetGroupListByUser(userid string) ([]*types.Group, error) {
	return nil, nil
}
func (m *mockArchiveGroupStore) GetGroupMemberByGroupID(groupId string) ([]*types.User, error) {
	return nil, nil
}
func (m *mockArchiveGroupStore) UpdateGroupMember(action string, userid string, groupid string) error {
	return nil
}
func (m *mockArchiveGroupStore) UpdateGroupStatus(groupid string, isActive bool) error {
	return nil
}
func (m *mockArchiveGroupStore) GetGroupCurrency(groupID string) (string, error) {
	return "", nil
}
func (m *mockArchiveGroupStore) GetRelatedUser(currentUser string) ([]*types.GroupMember, error) {
	return nil, nil
}

type mockArchiveGroupUserStore struct{}

func (m *mockArchiveGroupUserStore) GetUserByEmail(email string) (*types.User, error) {
	return nil, nil
}
func (m *mockArchiveGroupUserStore) GetUserByUsername(username string) (*types.User, error) {
	return nil, nil
}
func (m *mockArchiveGroupUserStore) GetUserByID(id string) (*types.User, error) {
	return nil, nil
}
func (m *mockArchiveGroupUserStore) CreateUser(user types.User) error {
	return nil
}
func (m *mockArchiveGroupUserStore) GetUsernameByID(userid string) (string, error) {
	return "", nil
}
func (m *mockArchiveGroupUserStore) CheckEmailExist(email string) (bool, error) {
	return false, nil
}
