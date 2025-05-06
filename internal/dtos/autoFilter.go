package dtos

type AutoFilter struct {
	Brand    string   `json:"brand"`
	Model    string   `json:"model"`
	MinYear  *uint32  `json:"min_year"`
	MaxYear  *uint32  `json:"max_year"`
	MinPrice *float64 `json:"min_price"`
	MaxPrice *float64 `json:"max_price"`
}

type AutoFilterResponse struct {
	Title    string  `json:"title"`
	Price    float64 `json:"price"`
	URL      string  `json:"url"`
	ImageURL string  `json:"image_url"`
}
