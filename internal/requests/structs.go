package requests

type Publisher struct {
	URL         string `json:"url" validate:"required"`
	Description string `json:"description"`
	Email       string `json:"email" validate:"email"`
}

type PublisherUpdate struct {
	URLAddresses []URLAddresses `json:"urlAddresses" validate:"required"`
	Description  string         `json:"description"`
	Email        string         `json:"email" validate:"email"`
}

type URLAddresses struct {
	URL string `json:"url" validate:"required"`
}

type Log struct {
	Message string `json:"message" validate:"required"`
}
