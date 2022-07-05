package requests

type Publisher struct {
	URL string `json:"url" validate:"required"`
}
