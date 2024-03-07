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
	default:
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
		&models.Software{},
		&models.SoftwareURL{},
		&models.Webhook{},
	); err != nil {
		return nil, fmt.Errorf("can't migrate database: %w", err)
	}

	// Migrate logs only if there is no "entity" column yet, which should mean when the database
	// is empty.
	// This is a workaround for https://github.com/go-gorm/gorm/issues/5534 where GORM
	// fails to migrate an existing generated column on PostgreSQL if it already exists.
	var entity string
	if database.Raw("SELECT entity FROM logs LIMIT 1").Scan(&entity).Error != nil {
		if err = database.AutoMigrate(&models.Log{}); err != nil {
			return nil, fmt.Errorf("can't migrate database: %w", err)
		}
	}

	return database, nil
}
