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
	OrganizationID string `json:"organizationId"`
	URL            string `json:"url"`
	Email          string `json:"email"`
}

type Software struct {
	gorm.Model
	Name string
}
