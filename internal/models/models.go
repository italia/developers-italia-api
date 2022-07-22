package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

type Model interface {
	TableName() string
	Uuid() string
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
	CreatedAt   time.Time      `json:"createdAt" gorm:"index"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

func (Publisher) TableName() string {
	return "publishers"
}

func (publisher Publisher) Uuid() string {
	return publisher.ID
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
	CreatedAt     time.Time      `json:"createdAt" gorm:"index"`
	UpdatedAt     time.Time      `json:"updatedAt"`
	DeletedAt     gorm.DeletedAt `json:"-" gorm:"index"`
}

func (Software) TableName() string {
	// Don't use GORM's default pluralized form ("softwares")
	return "software"
}

func (software Software) Uuid() string {
	return software.ID
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
	ID        string         `gorm:"primarykey"`
	URL       string         `json:"url" gorm:"uniqueIndex"`
	Secret    string         `json:"-"`
	CreatedAt time.Time      `json:"createdAt" gorm:"index"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `gorm:"index"`

	// TODO: composite unique index
	// Entity this Webhook is for (fe. Publisher, Software, etc.)
	EntityID   string `json:"-"`
	EntityType string `json:"-"`
}
