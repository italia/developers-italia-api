package models

import "gorm.io/gorm"

func Active(db *gorm.DB) *gorm.DB {
	return db.Where("active = ?", true)
}
