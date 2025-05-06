package services

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strconv"
	"strings"
	"time"

	"github.com/diegoafg1009/auto-radar-scraping-microservice/internal/dtos"
	"github.com/go-rod/rod"
	"github.com/go-rod/rod/lib/launcher"
)

const (
	neoAutoRodURL       = "https://www.neoauto.com/"
	neoAutoRodSearchURL = neoAutoRodURL + "venta-de-autos-usados"
	loaderImageURL      = "https://cds.neoauto.pe/neoauto3/img/loader_black.gif"
)

type NeoAutoRodscraper struct {
	baseURL   string
	searchURL string
}

func NewNeoAutoRodscraper() *NeoAutoRodscraper {
	return &NeoAutoRodscraper{
		baseURL:   neoAutoRodURL,
		searchURL: neoAutoRodSearchURL,
	}
}

func (s *NeoAutoRodscraper) FindByFilter(filter dtos.AutoFilter) ([]*dtos.AutoFilterResponse, error) {
	autos := make([]*dtos.AutoFilterResponse, 0)

	// Scrape with rod
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

			title, err := s.getCarTitle(content)
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
				URL:      s.baseURL + *url,
				ImageURL: imageURL,
				Price:    price,
			})

		}

		return nil
	}).MustDo()

	log.Println("Found", len(autos), "cars")

	return autos, nil
}

func (s *NeoAutoRodscraper) generateURL(filter dtos.AutoFilter) string {
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

	// Construir los parámetros de consulta
	params := make([]string, 0)

	if filter.MinYear != nil && *filter.MinYear > 0 {
		params = append(params, fmt.Sprintf("anio_min=%d", *filter.MinYear))
	}

	if filter.MaxYear != nil && *filter.MaxYear > 0 {
		params = append(params, fmt.Sprintf("anio_max=%d", *filter.MaxYear))
	}

	if filter.MinPrice != nil && *filter.MinPrice > 0 {
		params = append(params, fmt.Sprintf("precio_min=%.0f", *filter.MinPrice))
	}

	if filter.MaxPrice != nil && *filter.MaxPrice > 0 {
		params = append(params, fmt.Sprintf("precio_max=%.0f", *filter.MaxPrice))
	}

	// Si hay parámetros, añadirlos a la URL
	if len(params) > 0 {
		searchURL += "?" + strings.Join(params, "&")
	}

	return searchURL
}

func (s *NeoAutoRodscraper) getCarTitle(carResultContent *rod.Element) (string, error) {
	titleSelector := "div.c-results__header > h2"

	titleHeader, err := carResultContent.Element(titleSelector)
	if err != nil {
		return "", err
	}

	title, err := titleHeader.Text()
	if err != nil {
		return "", err
	}

	return title, nil
}

func (s *NeoAutoRodscraper) getCarImageURL(carResultBody *rod.Element) (string, error) {
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

func (s *NeoAutoRodscraper) retryGetImageURL(element *rod.Element, selector string, ctx context.Context) (string, error) {
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
func (s *NeoAutoRodscraper) getImageURL(element *rod.Element, selector string) (string, error) {
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

func (s *NeoAutoRodscraper) getCarPrice(carResultDetailContact *rod.Element) (price float64, err error) {
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

func (s *NeoAutoRodscraper) parsePriceFromText(textPrice string) (price float64, err error) {
	textPrice = strings.ReplaceAll(textPrice, ",", "")
	splitedTextPrice := strings.Split(textPrice, " ")
	price, err = strconv.ParseFloat(splitedTextPrice[len(splitedTextPrice)-1], 64)
	if err != nil {
		return 0, err
	}

	return price, nil
}
