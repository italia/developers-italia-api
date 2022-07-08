package requests

type Publisher struct {
	URL         string `json:"url" validate:"required"`
	Description string `json:"description"`
	Email       string `json:"email" validate:"email"`
}
