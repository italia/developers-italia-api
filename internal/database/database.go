package database

import (
	"errors"
	"log"
	"strings"

	"github.com/jackc/pgconn"

	"github.com/italia/developers-italia-api/internal/common"
	"github.com/jackc/pgerrcode"
	"gorm.io/gorm"
)

const (
	uniqueConstraintFailedErrorSQLite = "UNIQUE constraint failed"
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
	if strings.Contains(dbError.Error(), uniqueConstraintFailedErrorSQLite) {
		return common.ErrDBUniqueConstraint
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
