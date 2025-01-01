package frontend

import (
	"context"
	"expense-tracker/config"
	"expense-tracker/frontend"
	"expense-tracker/frontend/handlers/auth"
	"expense-tracker/frontend/handlers/expense"
	"expense-tracker/frontend/handlers/group"
	"expense-tracker/frontend/handlers/index"
	"expense-tracker/frontend/handlers/users"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type FrontendServer struct {
	addr string

	engine *http.Server
}

func NewFrontendServer(addr string) *FrontendServer {
	return &FrontendServer{
		addr: addr,
	}
}

func (s *FrontendServer) Run() error {
	gin.SetMode(config.Envs.Mode)

	router := gin.New()
	router.Use(gin.Logger())

	// embed public folder
	router.StaticFS("/public", frontend.Static())

	// register frontend services
	authHandler := auth.NewHandler()
	authHandler.RegisterRoutes(router)

	protected := router.Group("")
	protected.Use(auth.JWTAuthMiddleware())

	indexHandler := index.NewHandler()
	indexHandler.RegisterRoutes(protected)

	userHandler := users.NewHandler()
	userHandler.RegisterRoutes(protected)

	groupHandler := group.NewHandler()
	groupHandler.RegisterRoutes(protected)

	expenseHandler := expense.NewHandler()
	expenseHandler.RegisterRoutes(protected)

	log.Println("Frontend Server Listening on", s.addr)

	s.engine = &http.Server{
		Addr:    s.addr,
		Handler: router,
	}

	return s.engine.ListenAndServe()
}

func (s *FrontendServer) Shutdown(ctx context.Context) error {
	return s.engine.Shutdown(ctx)
}
