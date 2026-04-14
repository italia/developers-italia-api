package common

import (
	"strings"

	"github.com/PuerkitoBio/purell"
)

const normalizeURLFlags = purell.FlagsUsuallySafeGreedy | purell.FlagRemoveWWW

func NormalizeURL(rawURL string) string {
	normalized, err := purell.NormalizeURLString(rawURL, normalizeURLFlags)
	if err != nil {
		return rawURL
	}

	return normalized
}

type SourceInput struct {
	Driver *string  `json:"driver" validate:"omitempty,min=1,max=64"`
	URL    string   `json:"url" validate:"required,url"`
	Args   []string `json:"args" validate:"omitempty,dive,min=1"`
}

type CatalogPost struct {
	Name                string        `json:"name" validate:"required,min=1,max=255"`
	AlternativeID       *string       `json:"alternativeId" validate:"omitempty,min=1,max=255"`
	Active              *bool         `json:"active"`
	PublishersNamespace *string       `json:"publishersNamespace" validate:"omitempty,max=255"`
	Sources             []SourceInput `json:"sources" validate:"required,gt=0,dive"`
}

type CatalogPatch struct {
	Name                *string        `json:"name" validate:"omitempty,min=1,max=255"`
	AlternativeID       *string        `json:"alternativeId" validate:"omitempty,max=255"`
	Active              *bool          `json:"active"`
	PublishersNamespace *string        `json:"publishersNamespace" validate:"omitempty,max=255"`
	Sources             *[]SourceInput `json:"sources" validate:"omitempty,gt=0,dive"`
}

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
