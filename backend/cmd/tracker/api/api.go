package api

import (
	"context"
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
	"log"
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

type APIServer struct {
	addr string
	db   *sql.DB

	engine *http.Server
}

func NewAPIServer(addr string, db *sql.DB) *APIServer {
	return &APIServer{
		addr: addr,
		db:   db,
	}
}

func newBaseRouter() *gin.Engine {
	router := gin.New()
	if config.Envs.Mode != "release" {
		router.Use(gin.Logger())
	}
	router.Use(gin.Recovery(), middleware.CORSMiddleware())
	return router
}

func registerRoutes(router *gin.Engine, db *sql.DB) {
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

func newHTTPServer(addr string, handler http.Handler) *http.Server {
	return &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadHeaderTimeout: readHeaderTimeout,
		ReadTimeout:       readTimeout,
		WriteTimeout:      writeTimeout,
		IdleTimeout:       idleTimeout,
	}
}

func (s *APIServer) Run() error {
	gin.SetMode(config.Envs.Mode)

	router := newBaseRouter()
	registerRoutes(router, s.db)

	log.Println("API Server Listening on", s.addr)

	s.engine = newHTTPServer(s.addr, router)

	return s.engine.ListenAndServe()
}

func (s *APIServer) Shutdown(ctx context.Context) error {
	return s.engine.Shutdown(ctx)
}
