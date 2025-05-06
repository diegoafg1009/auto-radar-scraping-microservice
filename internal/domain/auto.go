package domain

type Auto struct {
	Model string  `json:"model"`
	Brand string  `json:"brand"`
	Year  uint32  `json:"year"`
	Price float64 `json:"price"`
	Image string  `json:"image"`
	Url   string  `json:"url"`
}
