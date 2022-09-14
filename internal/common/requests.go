package common

import "strings"

type Publisher struct {
	CodeHosting  []CodeHosting `json:"codeHosting" validate:"required,gt=0,dive"`
	Description  string        `json:"description"`
	Email        string        `json:"email" validate:"email,required"`
	Active       *bool         `json:"active"`
	ExternalCode string        `json:"externalCode" validate:"max=255"`
}

type PublisherPost struct {
	CodeHosting  []CodeHosting `json:"codeHosting" validate:"required,gt=0,dive"`
	Description  string        `json:"description"`
	Email        string        `json:"email" validate:"email,required"`
	Active       *bool         `json:"active"`
	ExternalCode string        `json:"externalCode" validate:"max=255"`
}

type PublisherPatch struct {
	CodeHosting  []CodeHosting `json:"codeHosting" validate:"gt=0"`
	Description  string        `json:"description"`
	Email        string        `json:"email" validate:"email"`
	Active       *bool         `json:"active"`
	ExternalCode string        `json:"externalCode" validate:"max=255"`
}

type CodeHosting struct {
	URL   string `json:"url" validate:"required,url"`
	Group *bool  `json:"group"`
}

type SoftwarePost struct {
	URL           string   `json:"url" validate:"required,url"`
	Aliases       []string `json:"aliases" validate:"dive,url"`
	PubliccodeYml string   `json:"publiccodeYml" validate:"required"`
	Active        *bool    `json:"active"`
}

type SoftwarePatch struct {
	URL           string    `json:"url" validate:"url"`
	Aliases       *[]string `json:"aliases" validate:"omitempty,dive,url"`
	PubliccodeYml string    `json:"publiccodeYml"`
	Active        *bool     `json:"active"`
}

type Log struct {
	Message string `json:"message" validate:"required,gt=1"`
}

type Webhook struct {
	URL    string `json:"url" validate:"required,url"`
	Secret string `json:"secret"`
}

func NormalizeEmail(email string) string {
	return strings.TrimSpace(strings.ToLower(email))
}
