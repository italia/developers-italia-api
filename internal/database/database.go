package database

import (
	"fmt"
	"log"
	"strings"

	"github.com/italia/developers-italia-api/internal/models"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func NewDatabase(connection string) (*gorm.DB, error) {
	var (
		database *gorm.DB
		err      error
	)

	switch {
	case strings.HasPrefix(connection, "file:"):
		log.Println("using SQLite database")

		database, err = gorm.Open(sqlite.Open(connection), &gorm.Config{TranslateError: true})
	case strings.HasPrefix(connection, "postgres:"):
		log.Println("using Postgres database")

		database, err = gorm.Open(postgres.Open(connection), &gorm.Config{
			TranslateError: true,
			PrepareStmt:    true,
			// Disable logging in production
			Logger: logger.Default.LogMode(logger.Silent),
		})
	}

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
