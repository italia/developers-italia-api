package common

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"
)

// pgConstraintToAPI maps PostgreSQL unique index names to API field names.
// GORM generates these as idx_<table>_<column> for uniqueIndex fields.
var pgConstraintToAPI = map[string]string{ //nolint:gochecknoglobals
	"idx_publishers_description":      "description",
	"idx_publishers_alternative_id":   "alternativeId",
	"idx_publishers_code_hosting_url": "codeHosting.url",
	"idx_software_urls_url":           "url",
}

// sqliteColToAPI maps SQLite "table.column" identifiers to API field names.
// SQLite unique constraint errors always have the format:
// "UNIQUE constraint failed: table.column".
var sqliteColToAPI = map[string]string{ //nolint:gochecknoglobals
	"publishers.description":      "description",
	"publishers.alternative_id":   "alternativeId",
	"publishers_code_hosting.url": "codeHosting.url",
	"software_urls.url":           "url",
}

// DuplicateField reports whether err is a unique constraint violation.
// Returns nil if it is not. Returns a pointer to the API field name that caused
// it (e.g. "alternativeId", "codeHosting.url") if it is, or a pointer to an
// empty string if the field cannot be determined.
func DuplicateField(err error) *string {
	if field := duplicateFieldSQLite(err); field != nil {
		return field
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code != "23505" {
			return nil
		}

		field := pgConstraintToAPI[pgErr.ConstraintName]

		return &field
	}

	return nil
}
