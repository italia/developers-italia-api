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
	ID   string `gorm:"primarykey"`
	Name string
}

type Log struct {
	ID        string         `json:"id" gorm:"primaryKey"`
	Message   string         `json:"message" gorm:"not null"`
	CreatedAt time.Time      `json:"createdAt" gorm:"index"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// Entity this Log entry is about (fe. Publisher, Software, etc.)
	EntityID   string `json:"-"`
	EntityType string `json:"-"`
}

type Publisher struct {
	ID          string         `gorm:"primarykey"`
	Email       string         `json:"email"`
	Description string         `json:"description"`
	CodeHosting []CodeHosting  `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;unique" json:"codeHosting"`
	Active      bool           `json:"active" gorm:"default:true"`
	CreatedAt   time.Time      `json:"createdAt" gorm:"index"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (Publisher) TableName() string {
	return "publishers"
}

func (p Publisher) UUID() string {
	return p.ID
}

func (p Publisher) AfterSave(tx *gorm.DB) error {
	return nil
}

type CodeHosting struct {
	gorm.Model
	URL         string `json:"url" gorm:"not null"`
	PublisherID string `json:"publisherId"`
}

type Software struct {
	ID            string         `json:"id" gorm:"primarykey"`
	URLs          []SoftwareURL  `json:"urls"`
	PubliccodeYml string         `json:"publiccodeYml"`
	Logs          []Log          `json:"-" gorm:"polymorphic:Entity;"`
	Active        bool           `json:"active" gorm:"default:true"`
	CreatedAt     time.Time      `json:"createdAt" gorm:"index"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Software) TableName() string {
	// Don't use GORM's default pluralized form ("softwares")
	return "software"
}

func (s Software) UUID() string {
	return s.ID
}

type SoftwareURL struct {
	gorm.Model
	ID         string `gorm:"primarykey"`
	URL        string `gorm:"uniqueIndex"`
	SoftwareID string
}

func (su SoftwareURL) MarshalJSON() ([]byte, error) {
	return ([]byte)(fmt.Sprintf(`"%s"`, su.URL)), nil
}

type Webhook struct {
	ID        string         `json:"id" gorm:"primarykey"`
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
	ID         string `gorm:"primarykey"`
	Type       string
	EntityType string
	EntityID   string
	CreatedAt  time.Time
	UpdatedAt  time.Time
	DeletedAt  gorm.DeletedAt `gorm:"index"`
}
