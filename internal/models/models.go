package models

import (
	"time"

	"gorm.io/gorm"
)

type Bundle struct {
	gorm.Model
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
	gorm.Model
	Name string `json:"name"`
}

type Software struct {
	gorm.Model
	Name string
}
