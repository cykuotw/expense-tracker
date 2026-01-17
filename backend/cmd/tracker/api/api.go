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

	"github.com/gin-gonic/gin"
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

func (s *APIServer) Run() error {
	gin.SetMode(config.Envs.Mode)

	router := gin.New()
	if config.Envs.Mode != "release" {
		router.Use(gin.Logger())
	}
	router.Use(middleware.CORSMiddleware())

	public := router.Group(config.Envs.APIPath)

	userStore := user.NewStore(s.db)
	userHandler := user.NewHandler(userStore)
	userHandler.RegisterRoutes(public)
	invitationStore := invitation.NewStore(s.db)
	invitationHandler := invitation.NewHandler(invitationStore)
	refreshStore := auth.NewRefreshStore(s.db)

	authHandler := authRoute.NewHandler(userStore, invitationStore, refreshStore)
	authHandler.RegisterRoutes(public)

	protected := public.Group("")
	protected.Use(auth.JWTAuthMiddleware())

	adminProtected := protected.Group("")
	adminProtected.Use(middleware.AdminMiddleware(userStore))
	invitationHandler.RegisterRoutes(public, adminProtected)

	userProtectedHandler := user.NewProtectedHandler(userStore)
	userProtectedHandler.RegisterRoutes(protected)

	groupStore := groupStore.NewStore(s.db)
	groupHandler := groupRoute.NewHandler(groupStore, userStore)
	groupHandler.RegisterRoutes(protected)

	expenseStore := expenseStore.NewStore(s.db)
	expenseController := expense.NewController()
	expenseHandler := expenseRoute.NewHandler(expenseStore, userStore, groupStore, expenseController)
	expenseHandler.RegisterRoutes(protected)

	log.Println("API Server Listening on", s.addr)

	s.engine = &http.Server{
		Addr:    s.addr,
		Handler: router,
	}

	return s.engine.ListenAndServe()
}

func (s *APIServer) Shutdown(ctx context.Context) error {
	return s.engine.Shutdown(ctx)
}
