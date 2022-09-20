package database

import (
	"errors"
	"log"

	"github.com/jackc/pgconn"
	"github.com/mattn/go-sqlite3"

	"github.com/italia/developers-italia-api/internal/common"
	"github.com/jackc/pgerrcode"
	"gorm.io/gorm"
)

type Database interface {
	Init(dsn string) (*gorm.DB, error)
}

//nolintlint:ireturn
func NewDatabase(env common.Environment) Database {
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

//nolint:errorlint
func WrapErrors(dbError error) error {
	if e, ok := dbError.(sqlite3.Error); ok {
		if e.ExtendedCode == sqlite3.ErrConstraintUnique {
			return common.ErrDBUniqueConstraint
		}
	}

	if e, ok := dbError.(*pgconn.PgError); ok {
		if e.Code == pgerrcode.UniqueViolation {
			return common.ErrDBUniqueConstraint
		}
	}

	if errors.Is(dbError, gorm.ErrRecordNotFound) {
		return common.ErrDBRecordNotFound
	}

	return dbError
}
