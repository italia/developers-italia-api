package database

import (
	"log"

	"github.com/italia/developers-italia-api/internal/common"
	"gorm.io/gorm"
)

type Database interface {
	Init(dsn string) (*gorm.DB, error)
}

//nolintlint:ireturn
func NewDatabase(env common.Environment) Database {
	log.Print(env)

	if env.IsTest() {
		log.Println("using SQLite database")

		return &SQLiteDB{
			dsn: env.Database,
		}
	}

	log.Println("using Postgres database")

	return &PostgresDB{
		dsn: env.Database,
	}
}
