package requests

type Publisher struct {
	URL         string `json:"url" validate:"required"`
	Description string `json:"description"`
	Email       string `json:"email" validate:"email"`
}

type PublisherUpdate struct {
	URLAddresses []URLAddresses `json:"url_addresses" validate:"required"`
	Description  string         `json:"description"`
	Email        string         `json:"email" validate:"email"`
}

type URLAddresses struct {
	URL string `json:"url" validate:"required"`
}
