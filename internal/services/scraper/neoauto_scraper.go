package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/diegoafg1009/auto-radar-scraping-microservice/internal"
	"github.com/diegoafg1009/auto-radar-scraping-microservice/internal/database"
	"github.com/diegoafg1009/auto-radar-scraping-microservice/internal/dtos"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

const (
	neoAutoURL       = "https://www.neoauto.com/"
	neoAutoSearchURL = neoAutoURL + "venta-de-autos-usados"
	loaderImageURL   = "https://cds.neoauto.pe/neoauto3/img/loader_black.gif"
)

type NeoAutoScraper struct {
	baseURL   string
	searchURL string
	db        database.Service
}

func NewNeoAutoScraper(db database.Service) *NeoAutoScraper {
	return &NeoAutoScraper{
		baseURL:   neoAutoURL,
		searchURL: neoAutoSearchURL,
		db:        db,
	}
}

func (s *NeoAutoScraper) FindByFilter(filter dtos.AutoFilter) ([]*dtos.AutoFilterResponse, error) {
	autos, alreadyFiltered, err := s.getAutos(filter)
	if err != nil {
		return nil, err
	}

	if alreadyFiltered {
		log.Println("Found", len(autos), "cars in cache")
		return autos, nil
	}

	autos = s.filterAutos(filter, autos)

	log.Println("Found", len(autos), "cars")

	complexFilteredRedisKey := generateComplexFilteredNeoAutosRedisKey(filter)

	s.saveAutosToCache(complexFilteredRedisKey, autos)

	return autos, nil
}

func (s *NeoAutoScraper) getAutos(filter dtos.AutoFilter) (autos []*dtos.AutoFilterResponse, alreadyFiltered bool, err error) {
	autos, alreadyFiltered, err = s.getAutosFromCache(filter)

	if err == nil {
		return autos, alreadyFiltered, nil
	}

	autos, err = s.scrapeAutos(filter)
	if err != nil {
		return nil, false, err
	}

	simpleFilteredRedisKey := generateSimpleFilteredNeoAutosRedisKey(filter)

	s.saveAutosToCache(simpleFilteredRedisKey, autos)

	return autos, alreadyFiltered, err
}

func (s *NeoAutoScraper) filterAutos(filter dtos.AutoFilter, autos []*dtos.AutoFilterResponse) []*dtos.AutoFilterResponse {
	filteredAutos := make([]*dtos.AutoFilterResponse, 0)

	for _, auto := range autos {
		if *filter.MinPrice != 0 && auto.Price < *filter.MinPrice {
			continue
		}
		if *filter.MaxPrice != 0 && auto.Price > *filter.MaxPrice {
			continue
		}
		if *filter.MinYear != 0 && auto.Year < *filter.MinYear {
			continue
		}
		if *filter.MaxYear != 0 && auto.Year > *filter.MaxYear {
			continue
		}
		filteredAutos = append(filteredAutos, auto)
	}

	return filteredAutos
}

func (s *NeoAutoScraper) getAutosFromCache(filter dtos.AutoFilter) (autos []*dtos.AutoFilterResponse, alreadyFiltered bool, err error) {
	complexFilteredRedisKey := generateComplexFilteredNeoAutosRedisKey(filter)
	err = s.db.GetJson(complexFilteredRedisKey, &autos)
	if err == nil {
		return autos, true, nil
	}

	simpleFilteredRedisKey := generateSimpleFilteredNeoAutosRedisKey(filter)
	err = s.db.GetJson(simpleFilteredRedisKey, &autos)
	if err != nil {
		return nil, false, err
	}

	return autos, false, nil
}

func (s *NeoAutoScraper) saveAutosToCache(redisKey string, autos []*dtos.AutoFilterResponse) error {
	err := s.db.SaveJsonWithTTL(redisKey, &autos, 1*time.Hour)
	if err != nil {
		return err
	}
	return nil
}

func (s *NeoAutoScraper) scrapeAutos(filter dtos.AutoFilter) ([]*dtos.AutoFilterResponse, error) {
	autos := make([]*dtos.AutoFilterResponse, 0)

	path, hasLauncher := launcher.LookPath()
	if !hasLauncher {
		return nil, errors.New("launcher not found")
	}

	u, err := launcher.New().Headless(true).Leakless(true).Bin(path).Launch()
	if err != nil {
		return nil, err
	}

	browser := rod.New().ControlURL(u).MustConnect()
	defer browser.MustClose()

	log.Println("Generating URL")

	searchURL := s.generateURL(filter)

	log.Println("Searching URL", searchURL)

	page := browser.MustPage(searchURL)
	defer page.MustClose()

	log.Println("Waiting for cars articles...")

	page.Race().Element("body > div.s-search > div.s-container > div.s-results.js-container.js-results-container").Handle(func(e *rod.Element) error {
		carsArticles, err := e.Elements("article")
		if err != nil {
			return err
		}

		var anchorHeight float64

		for _, carArticle := range carsArticles {
			anchorSelector := "a.c-results__link"

			anchor, err := carArticle.Element(anchorSelector)
			if err != nil {
				continue
			}

			if anchorHeight == 0 {
				anchorHeight = anchor.MustEval(`() => this.offsetHeight`).Num()
			}

			url := anchor.MustAttribute("href")
			if url == nil {
				continue
			}

			contentSelector := "div.c-results__content"

			content := carArticle.MustElement(contentSelector)

			title, year, err := s.getCarTitleAndYear(content)
			if err != nil {
				continue
			}

			resultBodySelector := "div.c-results__body"

			resultBody := content.MustElement(resultBodySelector)

			imageURL, err := s.getCarImageURL(resultBody)
			if err != nil {
				continue
			}

			// scroll based on anchor height
			page.MustEval(`() => {
				window.scrollBy(0, ` + strconv.FormatFloat(anchorHeight, 'f', -1, 64) + `)
			}`)

			contactSelector := "div.c-results-details__contact"

			contact := content.MustElement(contactSelector)

			price, err := s.getCarPrice(contact)
			if err != nil {
				continue
			}

			autos = append(autos, &dtos.AutoFilterResponse{
				Title:    title,
				Year:     year,
				URL:      s.baseURL + *url,
				ImageURL: imageURL,
				Price:    price,
			})

		}

		return nil
	}).MustDo()

	return autos, nil
}

func (s *NeoAutoScraper) generateURL(filter dtos.AutoFilter) string {
	searchURL := s.searchURL

	// Añadir filtro de marca y modelo si están especificados
	if filter.Brand != "" {
		// Normalizar la marca (convertir espacios en guiones, minúsculas)
		brand := strings.ToLower(strings.ReplaceAll(filter.Brand, " ", "-"))
		searchURL += "-" + brand

		if filter.Model != "" {
			// Normalizar el modelo
			model := strings.ToLower(strings.ReplaceAll(filter.Model, " ", "-"))
			searchURL += "-" + model
		}
	}

	return searchURL
}

func (s *NeoAutoScraper) getCarTitleAndYear(carResultContent *rod.Element) (title string, year uint32, err error) {
	titleSelector := "div.c-results__header > h2"

	titleHeader, err := carResultContent.Element(titleSelector)
	if err != nil {
		return "", 0, err
	}

	title, err = titleHeader.Text()
	if err != nil {
		return "", 0, err
	}

	title = strings.TrimSpace(title)

	// year is the last 4 characters of the title
	yearText := title[len(title)-4:]

	year, err = internal.StringToUint32(yearText)
	if err != nil {
		return "", 0, err
	}

	return title, year, nil
}

func (s *NeoAutoScraper) getCarImageURL(carResultBody *rod.Element) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	slidesContainer, err := carResultBody.Element("ul.glide__slides")
	if err != nil {
		return "", err
	}

	imageURL, err := s.retryGetImageURL(slidesContainer, "li.glide__slide--active > a", ctx)
	if err != nil {
		return "", err
	}

	return imageURL, nil
}

func (s *NeoAutoScraper) retryGetImageURL(element *rod.Element, selector string, ctx context.Context) (string, error) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			imageURL, err := s.getImageURL(element, selector)
			if err != nil {
				continue
			}

			if imageURL == loaderImageURL {
				continue
			}

			return imageURL, nil
		}
	}
}
func (s *NeoAutoScraper) getImageURL(element *rod.Element, selector string) (string, error) {
	imageContainer, err := element.Element(selector)

	if err != nil {
		return "", err
	}

	image, err := imageContainer.Element("img")

	if err != nil {
		return "", err
	}

	imageURL := image.MustAttribute("src")

	return *imageURL, nil
}

func (s *NeoAutoScraper) getCarPrice(carResultDetailContact *rod.Element) (price float64, err error) {
	priceSelector := "div.c-results-mount__price"

	textPrice := carResultDetailContact.MustElement(priceSelector).MustText()

	if textPrice == "" {
		priceSelector = "div.c-results-mount__santander-price"
		textPrice = carResultDetailContact.MustElement(priceSelector).MustText()
	}

	price, err = s.parsePriceFromText(textPrice)
	if err != nil {
		return 0, err
	}

	return price, nil

}

func (s *NeoAutoScraper) parsePriceFromText(textPrice string) (price float64, err error) {
	textPrice = strings.ReplaceAll(textPrice, ",", "")
	splitedTextPrice := strings.Split(textPrice, " ")
	price, err = strconv.ParseFloat(splitedTextPrice[len(splitedTextPrice)-1], 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}

func generateSimpleFilteredNeoAutosRedisKey(filter dtos.AutoFilter) string {
	return fmt.Sprintf("neo-auto:%s:%s", filter.Brand, filter.Model)
}

func generateComplexFilteredNeoAutosRedisKey(filter dtos.AutoFilter) string {
	minPrice := 0.0
	maxPrice := 0.0
	minYear := uint32(0)
	maxYear := uint32(0)

	if filter.MinPrice != nil {
		minPrice = *filter.MinPrice
	}
	if filter.MaxPrice != nil {
		maxPrice = *filter.MaxPrice
	}
	if filter.MinYear != nil {
		minYear = *filter.MinYear
	}
	if filter.MaxYear != nil {
		maxYear = *filter.MaxYear
	}

	return fmt.Sprintf("neo-auto:%s:%s:min-price:%.2f:max-price:%.2f:min-year:%d:max-year:%d",
		filter.Brand, filter.Model, minPrice, maxPrice, minYear, maxYear)
}
