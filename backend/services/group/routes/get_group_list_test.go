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

func TestGetGroupList(t *testing.T) {
	store := getGroupListStoreMock()
	userStore := getGroupListUserStoreMock()
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
		for _, group := range rsp {
			assert.Equal(t, types.GroupBalanceStatusSettled, group.BalanceStatus)
		}
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
		rawBody := bytes.TrimSpace(rr.Body.Bytes())

		var rsp []types.GetGroupListResponse
		err = json.NewDecoder(bytes.NewReader(rawBody)).Decode(&rsp)
		if err != nil {
			t.Fatal(err)
		}

		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(t, "[]", string(rawBody))
		assert.Equal(t, 0, len(rsp))
	})
}

var mockGroupListLen = 5

func TestGetGroupListReturnsStoreOrder(t *testing.T) {
	store := groupStoreMock()
	userStore := getGroupListUserStoreMock()
	handler := NewHandler(store, userStore)

	groupA := uuid.New()
	groupB := uuid.New()
	groupC := uuid.New()

	store.GetGroupListByUserFn = func(userid string) ([]types.GetGroupListResponse, error) {
		return []types.GetGroupListResponse{
			{ID: groupC.String(), GroupName: "Unsettled recent", BalanceStatus: types.GroupBalanceStatusOwed},
			{ID: groupB.String(), GroupName: "Unsettled older", BalanceStatus: types.GroupBalanceStatusOwing},
			{ID: groupA.String(), GroupName: "Settled recent", BalanceStatus: types.GroupBalanceStatusSettled},
		}, nil
	}

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
	assert.Len(t, rsp, 3)
	assert.Equal(t, groupC.String(), rsp[0].ID)
	assert.Equal(t, groupB.String(), rsp[1].ID)
	assert.Equal(t, groupA.String(), rsp[2].ID)
}
