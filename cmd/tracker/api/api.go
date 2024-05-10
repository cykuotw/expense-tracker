package api

import (
	"database/sql"
	"expense-tracker/services/auth"
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

	protected := subrouter.Group("/p")
	protected.Use(auth.JWTAuthMiddleware()) // use jwt middleware

	log.Println("Listening on", s.addr)

	return http.ListenAndServe(s.addr, router)
}
