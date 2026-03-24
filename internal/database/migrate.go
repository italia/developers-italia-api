package database

import (
	"errors"
	"net"
	"net/url"
	"strings"
)

// ErrEmptyDSN is returned when DATABASE_DSN is not set.
var ErrEmptyDSN = errors.New("DATABASE_DSN is not set")

// DSNToURL converts a lib/pq key=value DSN to a postgres:// URL suitable
// for the Atlas CLI. If dsn already starts with postgres:// or postgresql://
// it is returned unchanged.
//
// Limitation: in key=value format, values containing spaces are not supported
// (e.g. a password with a space). Use URL format for such cases.
func DSNToURL(dsn string) (string, error) {
	if dsn == "" {
		return "", ErrEmptyDSN
	}

	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		return dsn, nil
	}

	fields := strings.Fields(dsn)
	params := map[string]string{}

	for _, part := range fields {
		k, v, found := strings.Cut(part, "=")
		if !found {
			continue
		}

		params[k] = v
	}

	port := params["port"]
	if port == "" {
		port = "5432"
	}

	pgURL := &url.URL{
		Scheme: "postgres",
		User:   url.UserPassword(params["user"], params["password"]),
		Host:   net.JoinHostPort(params["host"], port),
		Path:   "/" + params["dbname"],
	}

	if sslmode := params["sslmode"]; sslmode != "" {
		pgURL.RawQuery = url.Values{"sslmode": {sslmode}}.Encode()
	}

	return pgURL.String(), nil
}
