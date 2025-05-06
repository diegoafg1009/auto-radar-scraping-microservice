package handlers

import (
	"context"

	"github.com/diegoafg1009/auto-radar-scraping-microservice/pkg/genproto/autoscrapper/v1/autoscrapperv1connect"

	v1 "github.com/diegoafg1009/auto-radar-scraping-microservice/pkg/genproto/autoscrapper/v1"

	services "github.com/diegoafg1009/auto-radar-scraping-microservice/internal/services/scraper"

	"github.com/diegoafg1009/auto-radar-scraping-microservice/internal/dtos"

	"connectrpc.com/connect"
)

type AutoScrapperHandler struct {
	autoscrapperv1connect.UnimplementedAutoScrapperServiceHandler
	autoscrapper services.AutoScrapper
}

func NewAutoScrapperHandler(autoscrapper services.AutoScrapper) *AutoScrapperHandler {
	return &AutoScrapperHandler{
		autoscrapper: autoscrapper,
	}
}

func (h *AutoScrapperHandler) FindByFilter(ctx context.Context, req *connect.Request[v1.FindByFilterRequest]) (*connect.Response[v1.FindByFilterResponse], error) {

	filter := dtos.AutoFilter{
		Brand:    req.Msg.Brand,
		Model:    req.Msg.Model,
		MinYear:  &req.Msg.MinYear,
		MaxYear:  &req.Msg.MaxYear,
		MinPrice: &req.Msg.MinPrice,
		MaxPrice: &req.Msg.MaxPrice,
	}

	autos, _ := h.autoscrapper.FindByFilter(filter)

	var autosResponse []*v1.Auto

	for _, auto := range autos {
		autosResponse = append(autosResponse, &v1.Auto{
			Title:    auto.Title,
			Price:    auto.Price,
			Url:      auto.URL,
			ImageUrl: auto.ImageURL,
		})
	}

	response := &v1.FindByFilterResponse{
		Autos: autosResponse,
	}

	return connect.NewResponse(response), nil
}
