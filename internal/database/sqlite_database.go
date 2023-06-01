package database

import (
	"fmt"

	"github.com/italia/developers-italia-api/internal/models"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type SQLiteDB struct {
	dsn string
}

func (d *SQLiteDB) Init(dsn string) (*gorm.DB, error) {
	var err error

	database, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		return nil, fmt.Errorf("can't open database: %w", err)
	}

	if err = database.AutoMigrate(
		&models.Publisher{},
		&models.Event{},
		&models.CodeHosting{},
		&models.Log{},
		&models.Software{},
		&models.SoftwareURL{},
		&models.Webhook{},
	); err != nil {
		return nil, fmt.Errorf("can't migrate database: %w", err)
	}

	return database, nil
}
