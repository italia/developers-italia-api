package database

import (
	"fmt"

	"github.com/italia/developers-italia-api/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type PostgresDB struct {
	dsn string
}

func (d *PostgresDB) Init(dsn string) (*gorm.DB, error) {
	var err error

	database, err := gorm.Open(postgres.Open(dsn), &gorm.Config{
		PrepareStmt: true,
	})
	if err != nil {
		return nil, fmt.Errorf("can't open database: %w", err)
	}

	if err = database.AutoMigrate(&models.Publisher{}, &models.URL{}); err != nil {
		return nil, fmt.Errorf("can't migrate database: %w", err)
	}

	return database, nil
}
