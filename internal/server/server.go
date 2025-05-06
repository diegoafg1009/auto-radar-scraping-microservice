package server

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"github.com/diegoafg1009/auto-radar-scraping-microservice/internal/database"
)

type Server struct {
	port      int
	db        database.Service
	apiServer *http.Server
}

func NewServer() *Server {
	port, _ := strconv.Atoi(os.Getenv("PORT"))
	NewServer := &Server{
		port: port,

		db: database.New(),
	}

	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", NewServer.port),
		Handler:      NewServer.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	NewServer.apiServer = server

	return NewServer
}

func (s *Server) ListenAndServe() error {
	log.Println("Starting server on port", s.port)
	return s.apiServer.ListenAndServe()
}

func (s *Server) Shutdown(ctx context.Context) error {
	log.Println("Shutting down server on port", s.port)
	return s.apiServer.Shutdown(ctx)
}
