package server

import (
	"net/http"

	"github.com/diegoafg1009/auto-radar-scraping-microservice/internal/handlers"
	services "github.com/diegoafg1009/auto-radar-scraping-microservice/internal/services/scraper"
	"github.com/diegoafg1009/auto-radar-scraping-microservice/pkg/genproto/autoscraper/v1/autoscraperv1connect"
	"golang.org/x/net/http2"
	"golang.org/x/net/http2/h2c"
)

func (s *Server) RegisterRoutes() http.Handler {
	mux := http.NewServeMux()

	path, handler := autoscraperv1connect.NewAutoScraperServiceHandler(handlers.NewAutoScraperHandler(services.NewNeoAutoRodscraper()))

	mux.Handle(path, handler)

	return h2c.NewHandler(mux, &http2.Server{})
}
