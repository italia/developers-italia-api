package models

import (
	"fmt"
	"time"

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
	EntityID   *string `json:"-"`
	EntityType *string `json:"-"`
	Entity     string  `json:"entity,omitempty" gorm:"->;type:text GENERATED ALWAYS AS (CASE WHEN entity_id IS NULL THEN NULL ELSE ('/' || entity_type || '/' || entity_id) END) STORED;default:(-);"` //nolint:lll
}

type Publisher struct {
	ID            string         `json:"id" gorm:"primaryKey"`
	Email         *string        `json:"email,omitempty"`
	Description   string         `json:"description" gorm:"uniqueIndex;not null"`
	CodeHosting   []CodeHosting  `json:"codeHosting" gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;unique"`
	Active        *bool          `json:"active" gorm:"default:true;not null"`
	AlternativeID *string        `json:"alternativeId,omitempty" gorm:"uniqueIndex"`
	CreatedAt     time.Time      `json:"createdAt" gorm:"index"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
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
	ID          string         `json:"-" gorm:"primaryKey"`
	URL         string         `json:"url" gorm:"not null;uniqueIndex"`
	Group       *bool          `json:"group" gorm:"default:true;not null"`
	PublisherID string         `json:"-"`
	CreatedAt   time.Time      `json:"createdAt" gorm:"index"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

type Software struct {
	ID string `json:"id" gorm:"primarykey"`

	// This needs to be explicitly declared, otherwise GORM won't create
	// the foreign key and will be confused about the double relationship
	// with SoftwareURLs (belongs to and has many).
	SoftwareURLID string `json:"-" gorm:"uniqueIndex;not null"`

	URL           SoftwareURL   `json:"url"`
	Aliases       []SoftwareURL `json:"aliases"`
	PubliccodeYml string        `json:"publiccodeYml"`
	Logs          []Log         `json:"-" gorm:"polymorphic:Entity;"`
	Active        *bool         `json:"active" gorm:"default:true;not null"`
	CreatedAt     time.Time     `json:"createdAt" gorm:"index"`
	UpdatedAt     time.Time     `json:"updatedAt"`
}

func (Software) TableName() string {
	// Don't use GORM's default pluralized form ("softwares")
	return "software"
}

func (s Software) UUID() string {
	return s.ID
}

type SoftwareURL struct {
	ID         string    `gorm:"primarykey"`
	URL        string    `gorm:"uniqueIndex"`
	SoftwareID string    `gorm:"not null"`
	CreatedAt  time.Time `json:"createdAt" gorm:"index"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

func (su SoftwareURL) MarshalJSON() ([]byte, error) {
	return ([]byte)(fmt.Sprintf(`"%s"`, su.URL)), nil
}

type Webhook struct {
	ID        string         `json:"id" gorm:"primaryKey"`
	URL       string         `json:"url" gorm:"index:idx_webhook_url,unique"`
	Secret    string         `json:"-"`
	CreatedAt time.Time      `json:"createdAt" gorm:"index"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

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
