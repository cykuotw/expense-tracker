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

func TestGetGroup(t *testing.T) {
	store := getGroupStoreMock()
	userStore := getGroupUserStoreMock()
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
