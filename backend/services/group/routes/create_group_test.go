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

func TestCreateGroup(t *testing.T) {
	store := createGroupStoreMock()
	userStore := createGroupUserStoreMock()
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
	t.Run("valid with selected members", func(t *testing.T) {
		memberID := uuid.New()
		var createdMemberIDs []string
		store.CreateGroupFn = func(_ types.Group, memberIDs []string) error {
			createdMemberIDs = memberIDs
			return nil
		}
		userStore.CheckUserExistByIDFn = func(id string) (bool, error) {
			return id == memberID.String(), nil
		}
		payload := types.CreateGroupPayload{
			GroupName: "testgroup",
			MemberIDs: []string{memberID.String(), memberID.String(), mockUserId.String()},
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
		assert.Equal(t, []string{memberID.String()}, createdMemberIDs)
	})
	t.Run("rejects an unknown selected member", func(t *testing.T) {
		unknownMemberID := uuid.NewString()
		createCalled := false
		store.CreateGroupFn = func(_ types.Group, _ []string) error {
			createCalled = true
			return nil
		}
		userStore.CheckUserExistByIDFn = func(string) (bool, error) {
			return false, nil
		}
		payload := types.CreateGroupPayload{
			GroupName: "testgroup",
			MemberIDs: []string{unknownMemberID},
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
		router := gin.New()
		router.POST("/create_group", handler.handleCreateGroup)
		router.ServeHTTP(rr, req)

		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.False(t, createCalled)
	})
}

var mockUserId = uuid.New()
