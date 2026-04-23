package models

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/italia/developers-italia-api/internal/common"
	"gorm.io/gorm"
)

type Model interface {
	TableName() string
	UUID() string
}

type Bundle struct {
	ID   string `gorm:"primaryKey"`
	Name string
}

type Log struct {
	ID        string         `json:"id" gorm:"primaryKey"`
	Message   string         `json:"message" gorm:"not null"`
	CreatedAt time.Time      `json:"createdAt" gorm:"index"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Entity this Log entry is about (fe. Publisher, Software, etc.)
	EntityID   *string `json:"-" gorm:"index:idx_log_entity"`
	EntityType *string `json:"-" gorm:"index:idx_log_entity"`
	Entity     string  `json:"entity,omitempty" gorm:"->;type:text GENERATED ALWAYS AS (CASE WHEN entity_id IS NULL THEN NULL ELSE ('/' || entity_type || '/' || entity_id) END) STORED;default:(-);"` //nolint:lll
}

type Catalog struct {
	ID                  string          `json:"id" gorm:"primaryKey"`
	Name                string          `json:"name" gorm:"not null"`
	AlternativeID       *string         `json:"alternativeId,omitempty" gorm:"uniqueIndex"`
	Active              *bool           `json:"active" gorm:"default:true;not null"`
	PublishersNamespace *string         `json:"publishersNamespace,omitempty"`
	Sources             []CatalogSource `json:"sources" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	CreatedAt           time.Time       `json:"createdAt" gorm:"index"`
	UpdatedAt           time.Time       `json:"updatedAt"`
}

type CatalogSource struct {
	ID        string    `json:"-" gorm:"primaryKey"`
	Driver    *string   `json:"driver,omitempty" gorm:"index"`
	URL       string    `json:"url" gorm:"not null"`
	Args      []string  `json:"args,omitempty" gorm:"serializer:json"`
	CatalogID string    `json:"-" gorm:"not null;index"`
	CreatedAt time.Time `json:"-" gorm:"index"`
	UpdatedAt time.Time `json:"-"`
}

func (CatalogSource) TableName() string {
	return "catalog_sources"
}

func (Catalog) TableName() string {
	return "catalogs"
}

func (c Catalog) UUID() string {
	return c.ID
}

type Publisher struct {
	ID            string        `json:"id" gorm:"primaryKey"`
	CatalogID     *string       `json:"catalogId,omitempty" gorm:"index"`
	Email         *string       `json:"email,omitempty"`
	Description   string        `json:"description" gorm:"uniqueIndex;not null"`
	CodeHosting   []CodeHosting `json:"codeHosting" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE"`
	Active        *bool         `json:"active" gorm:"default:true;not null"`
	AlternativeID *string       `json:"alternativeId,omitempty" gorm:"uniqueIndex"`
	CreatedAt     time.Time     `json:"createdAt" gorm:"index"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

func (Publisher) TableName() string {
	return "publishers"
}

func (p Publisher) UUID() string {
	return p.ID
}

func (CodeHosting) TableName() string {
	return "publishers_code_hosting"
}

type CodeHosting struct {
	ID          string    `json:"-" gorm:"primaryKey"`
	URL         string    `json:"url" gorm:"not null;uniqueIndex"`
	Group       *bool     `json:"group" gorm:"default:true;not null"`
	PublisherID string    `json:"-" gorm:"index"`
	CreatedAt   time.Time `json:"createdAt" gorm:"index"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Software struct {
	ID        string  `json:"id" gorm:"primarykey"`
	CatalogID *string `json:"catalogId,omitempty" gorm:"index"`

	// This needs to be explicitly declared, otherwise GORM won't create
	// the foreign key and will be confused about the double relationship
	// with SoftwareURLs (belongs to and has many).
	SoftwareURLID string `json:"-" gorm:"uniqueIndex;not null"`

	URL           SoftwareURL         `json:"url"`
	Aliases       SoftwareURLSlice    `json:"aliases"`
	PubliccodeYml string              `json:"publiccodeYml"`
	Logs          []Log               `json:"-" gorm:"polymorphic:Entity;"`
	Active        *bool               `json:"active" gorm:"default:true;not null"`
	Vitality      *string             `json:"vitality"`
	Analysis      common.AnalysisData `json:"-" gorm:"type:jsonb"`
	CreatedAt     time.Time           `json:"createdAt" gorm:"index"`
	UpdatedAt     time.Time           `json:"updatedAt"`
}

func (Software) TableName() string {
	// Don't use GORM's default pluralized form ("softwares")
	return "software"
}

func (s Software) UUID() string {
	return s.ID
}

//nolint:musttag // we are using a custom MarshalJSON method
type SoftwareURL struct {
	ID         string    `gorm:"primarykey"`
	URL        string    `gorm:"uniqueIndex"`
	SoftwareID string    `gorm:"not null;index"`
	CreatedAt  time.Time `gorm:"index"`
	UpdatedAt  time.Time
}

func (su SoftwareURL) MarshalJSON() ([]byte, error) {
	return fmt.Appendf(nil, `"%s"`, su.URL), nil
}

func (su *SoftwareURL) UnmarshalJSON(data []byte) error {
	//nolint:wrapcheck // we want to pass along the error here
	return json.Unmarshal(data, &su.URL)
}

type SoftwareURLSlice []SoftwareURL

func (slice SoftwareURLSlice) MarshalJSON() ([]byte, error) {
	urls := make([]string, len(slice))

	for i, su := range slice {
		urls[i] = su.URL
	}

	return json.Marshal(urls)
}

func (slice *SoftwareURLSlice) UnmarshalJSON(data []byte) error {
	var urls []string
	if err := json.Unmarshal(data, &urls); err != nil {
		//nolint:wrapcheck // we want to pass along the error here
		return err
	}

	// Convert each string URL into a SoftwareURL object.
	*slice = make(SoftwareURLSlice, len(urls))
	for i, urlStr := range urls {
		(*slice)[i] = SoftwareURL{URL: urlStr}
	}

	return nil
}

type Webhook struct {
	ID        string    `json:"id" gorm:"primaryKey"`
	URL       string    `json:"url" gorm:"index:idx_webhook_url,unique"`
	Secret    string    `json:"-"`
	CreatedAt time.Time `json:"createdAt" gorm:"index"`
	UpdatedAt time.Time `json:"updatedAt"`

	// Entity this Webhook is for (fe. Publisher, Software, etc.)
	EntityID   string `json:"-" gorm:"index:idx_webhook_url,unique"`
	EntityType string `json:"-" gorm:"index:idx_webhook_url,unique"`
}

type Event struct {
	ID         string `gorm:"primaryKey"`
	Type       string
	EntityType string
	EntityID   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}
