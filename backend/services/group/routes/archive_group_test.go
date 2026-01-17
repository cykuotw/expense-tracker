package group

import (
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func TestArchiveGroup(t *testing.T) {
	store := archiveGroupStoreMock()
	userStore := archiveGroupUserStoreMock()
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
