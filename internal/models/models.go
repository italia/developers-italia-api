package models

import (
	"time"

	"gorm.io/gorm"
)

type Bundle struct {
	ID   string `gorm:"primarykey"`
	Name string
}

type Log struct {
	ID        string `gorm:"primarykey"`
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}

type Publisher struct {
	ID          string `gorm:"primarykey"`
	Email       string `json:"email"`
	Description string `json:"description"`
	URL         []URL
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type URL struct {
	gorm.Model
	URL         string `json:"url"`
	PublisherID string
	CreatedAt   time.Time
	UpdatedAt   time.Time
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

type Software struct {
	ID        string `gorm:"primarykey"`
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
