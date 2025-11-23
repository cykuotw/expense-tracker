package api

import (
	"context"
	"database/sql"
	"expense-tracker/config"
	"expense-tracker/services/auth"
	authRoute "expense-tracker/services/auth/routes"
	"expense-tracker/services/expense"
	expenseRoute "expense-tracker/services/expense/routes"
	expenseStore "expense-tracker/services/expense/stores"
	groupRoute "expense-tracker/services/group/routes"
	groupStore "expense-tracker/services/group/stores"
	"expense-tracker/services/middleware"
	"expense-tracker/services/user"
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

	subrouter := router.Group(config.Envs.APIPath)

	userStore := user.NewStore(s.db)
	userHandler := user.NewHandler(userStore)
	userHandler.RegisterRoutes(subrouter)

	authHandler := authRoute.NewHandler(userStore)
	authHandler.RegisterRoutes(subrouter)

	protected := subrouter.Group("")
	protected.Use(auth.JWTAuthMiddleware())

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
