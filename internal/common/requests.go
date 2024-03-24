package common

import "strings"

type PublisherPost struct {
	CodeHosting   []CodeHosting `json:"codeHosting" validate:"required,gt=0,dive"`
	Description   string        `json:"description" validate:"required"`
	Email         *string       `json:"email" validate:"omitempty,email"`
	Active        *bool         `json:"active"`
	AlternativeID *string       `json:"alternativeId" validate:"omitempty,min=1,max=255"`
}

type PublisherPatch struct {
	CodeHosting   *[]CodeHosting `json:"codeHosting" validate:"omitempty,gt=0,dive"`
	Description   *string        `json:"description"`
	Email         *string        `json:"email" validate:"omitempty,email"`
	Active        *bool          `json:"active"`
	AlternativeID *string        `json:"alternativeId" validate:"omitempty,max=255"`
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
	Vitality      *string  `json:"vitality"`
}

type SoftwarePatch struct {
	URL           *string   `json:"url" validate:"omitempty,url"`
	Aliases       *[]string `json:"aliases" validate:"omitempty,dive,url"`
	PubliccodeYml *string   `json:"publiccodeYml"`
	Active        *bool     `json:"active"`
	Vitality      *string   `json:"vitality"`
}

type Log struct {
	Message string `json:"message" validate:"required,gt=1"`
}

type Webhook struct {
	URL    string `json:"url" validate:"required,url"`
	Secret string `json:"secret"`
}

func NormalizeEmail(email *string) *string {
	if email == nil {
		return nil
	}

	normalized := strings.TrimSpace(strings.ToLower(*email))

	return &normalized
}
