package tracker

import (
	"database/sql"
	"expense-tracker/backend/config"
	"expense-tracker/backend/services/auth"
	authRoute "expense-tracker/backend/services/auth/routes"
	"expense-tracker/backend/services/expense"
	expenseRoute "expense-tracker/backend/services/expense/routes"
	expenseStore "expense-tracker/backend/services/expense/stores"
	groupRoute "expense-tracker/backend/services/group/routes"
	groupStore "expense-tracker/backend/services/group/stores"
	"expense-tracker/backend/services/invitation"
	"expense-tracker/backend/services/middleware"
	"expense-tracker/backend/services/user"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
)

func NewHandler(db *sql.DB) http.Handler {
	gin.SetMode(config.Envs.Mode)

	router := gin.New()
	if config.Envs.Mode != "release" {
		router.Use(gin.Logger())
	}
	router.Use(gin.Recovery(), middleware.CORSMiddleware())

	registerRoutes(router, db)

	return router
}

func registerRoutes(router *gin.Engine, db *sql.DB) {
	router.GET(config.Envs.APIPath+"/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	public := router.Group(config.Envs.APIPath)
	public.Use(middleware.CSRFMiddleware())

	userStore := user.NewStore(db)
	userHandler := user.NewHandler(userStore)
	userHandler.RegisterRoutes(public)

	invitationStore := invitation.NewStore(db)
	invitationHandler := invitation.NewHandler(invitationStore)
	refreshStore := auth.NewRefreshStore(db)

	authHandler := authRoute.NewHandler(userStore, invitationStore, refreshStore)
	authHandler.RegisterRoutes(public)

	protected := public.Group("")
	protected.Use(auth.JWTAuthMiddleware())

	adminProtected := protected.Group("")
	adminProtected.Use(middleware.AdminMiddleware(userStore))
	invitationHandler.RegisterRoutes(public, adminProtected)

	userProtectedHandler := user.NewProtectedHandler(userStore)
	userProtectedHandler.RegisterRoutes(protected)

	groupStore := groupStore.NewStore(db)
	groupHandler := groupRoute.NewHandler(groupStore, userStore)
	groupHandler.RegisterRoutes(protected)

	expenseStore := expenseStore.NewStore(db)
	expenseController := expense.NewController()
	expenseHandler := expenseRoute.NewHandler(expenseStore, userStore, groupStore, expenseController)
	expenseHandler.RegisterRoutes(protected)
}

func NewHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}
