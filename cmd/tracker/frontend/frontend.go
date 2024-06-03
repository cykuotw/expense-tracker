package frontend

import (
	"context"
	frontend "expense-tracker/frontend/hanlders"
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
	router := gin.New()

	// register frontend services
	router.GET("/hello", frontend.Make(frontend.HandleHello))

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
