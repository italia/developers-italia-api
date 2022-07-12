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
	ID        uint           `json:"id" gorm:"primaryKey"`
	Message   string         `json:"message"`
	CreatedAt time.Time      `json:"createdAt" gorm:"index"`
	UpdatedAt time.Time      `json:"updatedAt"`
	DeletedAt gorm.DeletedAt `json:"deletedAt" gorm:"index"`
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

type CodeHosting struct {
	gorm.Model
	URL         string `json:"url" gorm:"not null"`
	PublisherID string `json:"publisherId"`
}

type Software struct {
	ID        string `gorm:"primarykey"`
	Name      string
	CreatedAt time.Time
	UpdatedAt time.Time
	DeletedAt gorm.DeletedAt `gorm:"index"`
}
