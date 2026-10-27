package tracker

import (
	"context"
	"database/sql"
	"expense-tracker/backend/config"
	"expense-tracker/backend/internal/observability"
	adminService "expense-tracker/backend/services/admin"
	"expense-tracker/backend/services/agenttools"
	"expense-tracker/backend/services/auth"
	authRoute "expense-tracker/backend/services/auth/routes"
	"expense-tracker/backend/services/expense"
	expenseRoute "expense-tracker/backend/services/expense/routes"
	expenseStore "expense-tracker/backend/services/expense/stores"
	"expense-tracker/backend/services/frontendreport"
	groupRoute "expense-tracker/backend/services/group/routes"
	groupStore "expense-tracker/backend/services/group/stores"
	"expense-tracker/backend/services/invitation"
	"expense-tracker/backend/services/middleware"
	"expense-tracker/backend/services/monthlyreview"
	"expense-tracker/backend/services/notification"
	"expense-tracker/backend/services/ocr"
	"expense-tracker/backend/services/user"
	"expense-tracker/internal/localonly"
	"fmt"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/s3"
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
	return newHandler(db, nil, options...)
}

// NewLocalHandler is an explicit standalone adapter. The deployed/default
// constructor never enables local tool access from an environment variable.
func NewLocalHandler(db *sql.DB, options ...HandlerOption) (http.Handler, error) {
	if err := ValidateLocalAgentMode(config.Envs); err != nil {
		return nil, err
	}
	tools, err := agenttools.NewLocalHandler(user.NewStore(db), groupStore.NewStore(db), expenseStore.NewStore(db))
	if err != nil {
		return nil, err
	}
	return newHandler(db, tools, options...), nil
}

func ValidateLocalAgentMode(cfg config.Config) error {
	if err := localonly.CheckRuntime(); err != nil {
		return err
	}
	if config.BuildMode == "release" || cfg.Mode == "release" {
		return fmt.Errorf("local agent tools are unavailable in release builds")
	}
	return localonly.CheckAddress(cfg.BackendURL)
}

func newHandler(db *sql.DB, localTools *agenttools.Handler, options ...HandlerOption) http.Handler {
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

	registerRoutes(router, db, localTools)

	return router
}

func registerRoutes(router *gin.Engine, db *sql.DB, localTools *agenttools.Handler) {
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
	ocr.NewCapabilityHandler(userStore, []byte(config.Envs.OCRCapabilitySecret)).RegisterRoutes(protected)

	groupStore := groupStore.NewStore(db)
	groupHandler := groupRoute.NewHandler(groupStore, userStore)
	groupHandler.RegisterRoutes(protected)

	notificationStore := notification.NewStore(db)
	notificationHandler := notification.NewHandler(notificationStore, config.Envs.WebPushVAPIDPublicKey)
	notificationHandler.RegisterRoutes(protected)
	monthlyreview.NewHandler(monthlyreview.NewStore(db)).RegisterRoutes(protected)
	frontendreport.NewHandler().RegisterRoutes(protected)

	expenseStore := expenseStore.NewStore(db)
	// Registered outside browser cookie/CSRF middleware. This read-only boundary
	// requires a dedicated JWT in both signed and explicit local transports.
	toolHandler := agenttools.NewHandler(userStore, groupStore, expenseStore, os.Getenv("AGENT_TOOL_ROLE_ARN"))
	if localTools != nil {
		toolHandler = localTools
	}
	toolHandler.RegisterRoutes(router.Group(config.Envs.APIPath))
	expenseController := expense.NewController()
	receiptOptions := []func(*expenseRoute.Handler){}
	if config.Envs.ReceiptBucket != "" {
		awsCfg, err := awsconfig.LoadDefaultConfig(context.Background())
		if err == nil {
			receiptOptions = append(receiptOptions, expenseRoute.WithReceiptStorage(
				s3.NewFromConfig(awsCfg), config.Envs.ReceiptBucket, []byte(config.Envs.OCRCapabilitySecret), config.Envs.ReceiptStorageEnabled))
		}
	}
	expenseHandler := expenseRoute.NewHandler(expenseStore, userStore, groupStore, expenseController, receiptOptions...)
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
