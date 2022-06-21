package database

import (
	"fmt"

	"github.com/italia/developers-italia-api/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

//nolint:gochecknoglobals // gorm suggests to do this in the examples
var Database *gorm.DB

func Init(dsn string) error {
	var err error

	Database, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
	})

	if err != nil {
		return fmt.Errorf("can't open database: %w", err)
	}

	if err = Database.AutoMigrate(&models.Publisher{}); err != nil {
		return fmt.Errorf("can't migrate database: %w", err)
	}

	return nil
}
