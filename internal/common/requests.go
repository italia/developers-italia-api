package common

type Publisher struct {
	CodeHosting []CodeHosting `json:"codeHosting" validate:"required"`
	Description string        `json:"description"`
	Email       string        `json:"email" validate:"email"`
}

type CodeHosting struct {
	URL string `json:"url" validate:"required"`
}

type Software struct {
	URLs          []string `json:"urls" validate:"required,gt=0"`
	PubliccodeYml string   `json:"publiccodeYml" validate:"required"`
}

type Log struct {
	Message string `json:"message" validate:"required,gt=1"`
}
