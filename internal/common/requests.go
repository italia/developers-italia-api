package common

type Publisher struct {
	URL         string `json:"url" validate:"required"`
	Description string `json:"description"`
	Email       string `json:"email" validate:"email"`
}

type PublisherUpdate struct {
	CodeHosting []CodeHosting `json:"codeHosting" validate:"required"`
	Description string        `json:"description"`
	Email       string        `json:"email" validate:"email"`
}

type CodeHosting struct {
	URL string `json:"url" validate:"required"`
}

type Software struct {
	URLs []string `json:"urls" validate:"required,gt=1"`
}

type Log struct {
	Message string `json:"message" validate:"required,gt=1"`
}
