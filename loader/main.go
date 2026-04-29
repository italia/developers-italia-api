// loader is an Atlas schema loader for the GORM models.
// It is used by Atlas to generate versioned SQL migrations.
//
// Usage: atlas migrate diff --env gorm
package main

import (
	"fmt"
	"io"
	"os"

	"ariga.io/atlas-provider-gorm/gormschema"
	"github.com/italia/developers-italia-api/internal/models"
)

func main() {
	stmts, err := gormschema.New("postgres").Load(
		&models.Publisher{},
		&models.CodeHosting{},
		&models.Software{},
		&models.SoftwareURL{},
		&models.Webhook{},
		&models.Event{},
		&models.Log{},
	)
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to load gorm schema: %v\n", err)
		os.Exit(1)
	}

	io.WriteString(os.Stdout, stmts) //nolint:errcheck
}
