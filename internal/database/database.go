package database

import (
	"fmt"

	"github.com/italia/developers-italia-api/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

var Database *gorm.DB

func Init(dsn string) error {
	var err error

	Database, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt:            true,
	})

	if err != nil {
		return fmt.Errorf("Can't open database: %s", err.Error())
	}

	if err = Database.AutoMigrate(&models.Publisher{}); err != nil {
		return fmt.Errorf("Can't migrate database: %s", err.Error())
	}

	return  nil
}
