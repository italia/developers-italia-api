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
	Name string `json:"name"`
}

type Software struct {
	gorm.Model
	Name string
}
