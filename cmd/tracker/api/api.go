package api

import (
	"database/sql"
	"expense-tracker/services/auth"
	"expense-tracker/services/expense"
	"expense-tracker/services/group"
	"expense-tracker/services/user"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
)

type APIServer struct {
	addr string
	db   *sql.DB
}

func NewAPIServer(addr string, db *sql.DB) *APIServer {
	return &APIServer{
		addr: addr,
		db:   db,
	}
}

func (s *APIServer) Run() error {
	router := gin.New()
	subrouter := router.Group("/api/v0")

	userStore := user.NewStore(s.db)
	userHandler := user.NewHandler(userStore)
	userHandler.RegisterRoutes(subrouter)

	protected := subrouter.Group("")
	protected.Use(auth.JWTAuthMiddleware())

	groupStore := group.NewStore(s.db)
	groupHandler := group.NewHandler(groupStore, userStore)
	groupHandler.RegisterRoutes(protected)

	expenseStore := expense.NewStore(s.db)
	expenseController := expense.NewController()
	expenseHandler := expense.NewHandler(expenseStore, userStore, groupStore, expenseController)
	expenseHandler.RegisterRoutes(protected)

	log.Println("Listening on", s.addr)

	return http.ListenAndServe(s.addr, router)
}
