package models

import (
	"time"

	"gorm.io/gorm"
)

type Bundle struct {
	ID   string `gorm:"primarykey"`
	Name string
}

type Log struct{}

type Publisher struct {
	ID           string         `gorm:"primarykey"`
	Email        string         `json:"email"`
	Description  string         `json:"description"`
	URLAddresses []URLAddresses `gorm:"constraint:OnUpdate:CASCADE,OnDelete:SET NULL;"`
	CreatedAt    time.Time      `json:"createdAt" gorm:"index"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
}

type URLAddresses struct {
	gorm.Model
	URL         string `json:"url"`
	PublisherID string
	CreatedAt   time.Time `json:"created_at"`
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
