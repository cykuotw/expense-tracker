package tracker

import (
	"database/sql"
	"expense-tracker/backend/config"
	"expense-tracker/backend/internal/observability"
	adminService "expense-tracker/backend/services/admin"
	"expense-tracker/backend/services/auth"
	authRoute "expense-tracker/backend/services/auth/routes"
	"expense-tracker/backend/services/expense"
	expenseRoute "expense-tracker/backend/services/expense/routes"
	expenseStore "expense-tracker/backend/services/expense/stores"
	groupRoute "expense-tracker/backend/services/group/routes"
	groupStore "expense-tracker/backend/services/group/stores"
	"expense-tracker/backend/services/invitation"
	"expense-tracker/backend/services/middleware"
	"expense-tracker/backend/services/notification"
	"expense-tracker/backend/services/user"
	"log/slog"
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	readHeaderTimeout = 5 * time.Second
	readTimeout       = 10 * time.Second
	writeTimeout      = 15 * time.Second
	idleTimeout       = 60 * time.Second
	slowRequest       = time.Second
)

type handlerConfig struct {
	logger *slog.Logger
}

type HandlerOption func(*handlerConfig)

func WithLogger(logger *slog.Logger) HandlerOption {
	return func(config *handlerConfig) {
		if logger != nil {
			config.logger = logger
		}
	}
}

func NewHandler(db *sql.DB, options ...HandlerOption) http.Handler {
	gin.SetMode(config.Envs.Mode)
	settings := handlerConfig{
		logger: observability.NewLogger(config.Envs.Mode, os.Stdout),
	}
	for _, option := range options {
		option(&settings)
	}

	router := gin.New()
	router.Use(observability.RequestLogging(observability.RequestLoggingConfig{
		Logger:               settings.logger,
		SlowRequestThreshold: slowRequest,
	}))
	router.Use(observability.Recovery(), middleware.CORSMiddleware())

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

	registrationStore := auth.NewRegistrationStore(db)
	authHandler := authRoute.NewHandler(userStore, invitationStore, refreshStore, registrationStore)
	authHandler.RegisterRoutes(public)

	protected := public.Group("")
	protected.Use(auth.JWTAuthMiddleware(), middleware.ActiveUserMiddleware(userStore))

	adminProtected := protected.Group("")
	adminProtected.Use(middleware.AdminMiddleware(userStore))
	adminHandler := adminService.NewHandler(userStore, invitationStore)
	adminHandler.RegisterRoutes(adminProtected)
	invitationHandler.RegisterRoutes(public, adminProtected)

	userProtectedHandler := user.NewProtectedHandler(userStore)
	userProtectedHandler.RegisterRoutes(protected)

	groupStore := groupStore.NewStore(db)
	groupHandler := groupRoute.NewHandler(groupStore, userStore)
	groupHandler.RegisterRoutes(protected)

	notificationStore := notification.NewStore(db)
	notificationHandler := notification.NewHandler(notificationStore, config.Envs.WebPushVAPIDPublicKey)
	notificationHandler.RegisterRoutes(protected)

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
