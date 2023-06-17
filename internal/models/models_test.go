package models

import (
	"database/sql"
	"log"
	"os"
	"testing"

	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/gofiber/fiber/v2/utils"
	_ "github.com/gofiber/fiber/v2/utils"
	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	sqlitedb *sql.DB
	db       *gorm.DB
)

func init() {
	os.Remove("./test.db")

	os.Setenv("DATABASE_DSN", "file:./test.db")
	os.Setenv("ENVIRONMENT", "test")

	dsn := os.Getenv("DATABASE_DSN")

	var err error
	sqlitedb, err = sql.Open("sqlite3", dsn)
	if err != nil {
		log.Fatal(err)
	}

	// This is needed, otherwise we get a database-locked error
	// TODO: investigate the root cause
	sqlitedb.Exec("PRAGMA journal_mode=WAL;")

	db, err = gorm.Open(sqlite.Open(dsn), &gorm.Config{TranslateError: true})
	if err != nil {
		log.Fatal(err)
	}

	if err = db.AutoMigrate(
		&Publisher{},
		&Event{},
		&CodeHosting{},
		&Log{},
		&Software{},
		&SoftwareURL{},
		&Webhook{},
	); err != nil {
		log.Fatal(err)
	}
}

func loadFixtures(t *testing.T, files ...string) {
	var filesWithPath []string
	for _, file := range files {
		filesWithPath = append(filesWithPath, "../../test/testdata/fixtures/"+file)
	}

	fixtures, err := testfixtures.New(
		testfixtures.Database(sqlitedb),
		testfixtures.Dialect("sqlite"),
		testfixtures.Files(filesWithPath...),
	)
	assert.Nil(t, err)

	err = fixtures.Load()
	assert.Nil(t, err)
}

func TestSoftwareCreate(t *testing.T) {
	loadFixtures(t, "software.yml", "software_urls.yml")

	// New ID
	err := db.Create(
		&Software{
			ID:            utils.UUIDv4(),
			URL:           SoftwareURL{ID: utils.UUIDv4(), URL: "https://example.org"},
			PubliccodeYml: "-",
		},
	).Error
	assert.NoError(t, err)

	// Duplicate ID
	err = db.Create(
		&Software{
			ID:            "c353756e-8597-4e46-a99b-7da2e141603b",
			URL:           SoftwareURL{ID: utils.UUIDv4(), URL: "https://example.org"},
			PubliccodeYml: "-",
		},
	).Error
	assert.ErrorIs(t, err, gorm.ErrDuplicatedKey)
}

func TestSoftwareURLCreate(t *testing.T) {
	loadFixtures(t, "software.yml", "software_urls.yml")

	// New ID
	err := db.Create(
		&SoftwareURL{
			ID:         utils.UUIDv4(),
			SoftwareID: "c353756e-8597-4e46-a99b-7da2e141603b",
			URL:        "https://new-1.example.org",
		},
	).Error
	assert.NoError(t, err)

	// Duplicate ID
	err = db.Create(
		&SoftwareURL{
			ID:         "beeadd3e-11bb-4313-99bb-94cd51836926",
			SoftwareID: "c353756e-8597-4e46-a99b-7da2e141603b",
			URL:        "https://new-2.example.org",
		},
	).Error
	assert.ErrorIs(t, err, gorm.ErrDuplicatedKey)
}

func TestPublisherCreate(t *testing.T) {
	loadFixtures(t, "publishers.yml")
	description := "New publisher"

	email := "new-publisher@example.org"

	// New ID
	err := db.Create(
		&Publisher{
			ID:          utils.UUIDv4(),
			Description: description,
			Email:       &email,
		},
	).Error
	assert.NoError(t, err)

	// Duplicate ID
	err = db.Create(
		&Publisher{
			ID:          "2ded32eb-c45e-4167-9166-a44e18b8adde",
			Description: description,
			Email:       &email,
		},
	).Error
	assert.ErrorIs(t, err, gorm.ErrDuplicatedKey)

	// Duplicate alternativeId
	alternativeID := "alternative-id-12345"
	err = db.Create(
		&Publisher{
			ID:            utils.UUIDv4(),
			Description:   "Another description",
			AlternativeID: &alternativeID,
		},
	).Error
	assert.ErrorIs(t, err, gorm.ErrDuplicatedKey)
}

func TestWebhookCreate(t *testing.T) {
	loadFixtures(t, "software.yml", "webhooks.yml")

	// New ID
	err := db.Create(
		&Webhook{
			ID:         utils.UUIDv4(),
			EntityID:   "c5dec6fa-8a01-4881-9e7d-132770d4214d",
			EntityType: "software",
			URL:        "https://new-webhook-1.example.org",
		},
	).Error
	assert.NoError(t, err)

	// Duplicate ID
	err = db.Create(
		&Webhook{
			ID:         "007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			EntityID:   "c5dec6fa-8a01-4881-9e7d-132770d4214d",
			EntityType: "software",
			URL:        "https://new-webhook-2.example.org",
		},
	).Error
	assert.ErrorIs(t, err, gorm.ErrDuplicatedKey)
}

func TestEventCreate(t *testing.T) {
	loadFixtures(t, "events.yml")

	// New ID
	err := db.Create(
		&Event{
			ID:         utils.UUIDv4(),
			Type:       "update",
			EntityID:   "c5dec6fa-8a01-4881-9e7d-132770d4214d",
			EntityType: "software",
		},
	).Error
	assert.NoError(t, err)

	// Duplicate ID
	err = db.Create(
		&Event{
			ID:         "d37d1082-528e-449d-a626-445561368d6b",
			Type:       "update",
			EntityID:   "c5dec6fa-8a01-4881-9e7d-132770d4214d",
			EntityType: "software",
		},
	).Error
	assert.ErrorIs(t, err, gorm.ErrDuplicatedKey)
}
