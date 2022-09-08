package common

type Publisher struct {
	CodeHosting  []CodeHosting `json:"codeHosting" validate:"required,gt=0,dive"`
	Description  string        `json:"description"`
	Email        string        `json:"email" validate:"email,required"`
	Active       bool          `json:"active"`
	ExternalCode string        `json:"externalCode" validate:"max=255"`
}

type CodeHosting struct {
	URL string `json:"url" validate:"required,url"`
}

type Software struct {
	URLs          []string `json:"urls" validate:"required,gt=0,dive,url"`
	PubliccodeYml string   `json:"publiccodeYml" validate:"required"`
	Active        bool     `json:"active"`
}

type Log struct {
	Message string `json:"message" validate:"required,gt=1"`
}

type Webhook struct {
	URL    string `json:"url" validate:"required,url"`
	Secret string `json:"secret"`
}
