package services

import (
	"github.com/diegoafg1009/auto-radar-scraping-microservice/internal/dtos"
)

type AutoScrapper interface {
	FindByFilter(filter dtos.AutoFilter) ([]*dtos.AutoFilterResponse, error)
}
