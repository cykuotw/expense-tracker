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
