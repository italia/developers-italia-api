package models

import "gorm.io/gorm"

type Bundle struct {
	gorm.Model
	Name string
}

type Log struct {
	gorm.Model
}

type Publisher struct {
	gorm.Model
	ID  string `json:"id"`
	URL string `json:"url"`
}

type Software struct {
	gorm.Model
	Name string
}
