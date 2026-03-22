//go:build cgo

package common

import (
	"errors"
	"strings"

	sqlite3 "github.com/mattn/go-sqlite3"
)

func duplicateFieldSQLite(err error) *string {
	var sqliteErr sqlite3.Error
	if !errors.As(err, &sqliteErr) {
		return nil
	}

	if sqliteErr.ExtendedCode != sqlite3.ErrConstraintUnique &&
		sqliteErr.ExtendedCode != sqlite3.ErrConstraintPrimaryKey {
		return nil
	}

	msg := sqliteErr.Error()
	if _, tableCol, ok := strings.Cut(msg, "UNIQUE constraint failed: "); ok {
		field := sqliteColToAPI[tableCol]

		return &field
	}

	empty := ""

	return &empty
}
