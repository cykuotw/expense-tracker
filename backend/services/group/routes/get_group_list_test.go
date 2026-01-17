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
