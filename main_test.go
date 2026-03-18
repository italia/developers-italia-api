package main

import (
	"database/sql"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	_ "github.com/lib/pq"
	_ "github.com/mattn/go-sqlite3"
)

const UUID_REGEXP = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

var (
	app       *fiber.App
	db        *sql.DB
	dbDriver  string
	goodToken = "Bearer v2.local.TwwHUQEi8hr2Eo881_Bs5vK9dHOR5BgEU24QRf-U7VmUwI1yOEA6mFT0EsXioMkFT_T-jjrtIJ_Nv8f6hR6ifJXUOuzWEkm9Ijq1mqSjQatD3aDqKMyjjBA"
	badToken  = "Bearer v2.local.UngfrCDNwGUw4pff2oBNoyxYvOErcbVVqLndl6nzONafUCzktaOeMSmoI7B0h62zoxXXLqTm_Phl"
)

type TestCase struct {
	description string

	// Test input
	query   string
	body    string
	headers map[string][]string

	// Expected output
	expectedCode        int
	expectedBody        string
	expectedContentType string
	validateFunc        func(t *testing.T, response map[string]interface{})
}

func init() {
	// Test on SQLite by default if DATABASE_DSN is not set
	if _, exists := os.LookupEnv("DATABASE_DSN"); !exists {
		_ = os.Setenv("DATABASE_DSN", "file:./test.db")
		_ = os.Remove("./test.db")
	}

	_ = os.Setenv("ENVIRONMENT", "test")

	// echo -n 'test-paseto-key-dont-use-in-prod'  | base64
	_ = os.Setenv("PASETO_KEY", "dGVzdC1wYXNldG8ta2V5LWRvbnQtdXNlLWluLXByb2Q=")

	dsn := os.Getenv("DATABASE_DSN")
	switch {
	case strings.HasPrefix(dsn, "postgres:"):
		dbDriver = "postgres"
	default:
		dbDriver = "sqlite3"
	}

	var err error
	db, err = sql.Open(dbDriver, dsn)
	if err != nil {
		log.Fatal(err)
	}

	// This is needed, otherwise we get a database-locked error
	// TODO: investigate the root cause
	if dbDriver == "sqlite3" {
		_, _ = db.Exec("PRAGMA journal_mode=WAL;")
	}

	// Setup the app as it is done in the main function
	app = Setup()
}

func TestMain(m *testing.M) {
	code := m.Run()

	os.Exit(code)
}

func loadFixtures(t *testing.T) {
	t.Helper()
	fixtures, err := testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Dialect(dbDriver),
		testfixtures.Directory("test/testdata/fixtures/"),
	)
	require.NoError(t, err, "failed to create test fixtures")

	err = fixtures.Load()
	require.NoError(t, err, "failed to load test fixtures")
}

func runTestCases(t *testing.T, tests []TestCase) {
	for _, test := range tests {
		description := test.description
		if description == "" {
			description = test.query
		}

		t.Run(description, func(t *testing.T) {
			loadFixtures(t)

			query := strings.Split(test.query, " ")

			u, err := url.Parse(query[1])
			if err != nil {
				assert.Fail(t, err.Error())
			}

			req, err := http.NewRequest(
				query[0],
				query[1],
				strings.NewReader(test.body),
			)
			if err != nil {
				assert.Fail(t, err.Error())
			}

			if test.headers != nil {
				req.Header = test.headers
			}
			req.URL.RawQuery = u.Query().Encode()

			res, err := app.Test(req, -1)
			assert.Nil(t, err)

			assert.Equal(t, test.expectedCode, res.StatusCode)

			body, err := io.ReadAll(res.Body)

			assert.Nil(t, err)

			if test.validateFunc != nil {
				var bodyMap map[string]interface{}
				err = json.Unmarshal(body, &bodyMap)
				require.NoError(t, err, "failed to unmarshal response body:\n%s", body)

				test.validateFunc(t, bodyMap)
				if t.Failed() {
					log.Printf("\nAPI response:\n%s\n", body)
				}
			} else {
				assert.Equal(t, test.expectedBody, string(body))
			}

			assert.Equal(t, test.expectedContentType, res.Header.Get("Content-Type"))
		})
	}
}

// assertUUID checks that val is a string matching the UUID format.
func assertUUID(t *testing.T, val interface{}) {
	t.Helper()
	s, ok := val.(string)
	require.True(t, ok, "expected string UUID, got %T: %v", val, val)
	match, err := regexp.MatchString(UUID_REGEXP, s)
	require.NoError(t, err)
	assert.True(t, match, "expected UUID format, got %q", s)
}

// assertRFC3339 checks that val is a string in RFC3339 format and returns the parsed time.
func assertRFC3339(t *testing.T, val interface{}) time.Time {
	t.Helper()
	s, ok := val.(string)
	require.True(t, ok, "expected RFC3339 string, got %T: %v", val, val)
	parsed, err := time.Parse(time.RFC3339, s)
	assert.NoError(t, err, "expected RFC3339 timestamp, got %q", s)
	return parsed
}

// assertTimestamps checks that the map has valid RFC3339 createdAt and updatedAt fields.
func assertTimestamps(t *testing.T, m map[string]interface{}) {
	t.Helper()
	assertRFC3339(t, m["createdAt"])
	assertRFC3339(t, m["updatedAt"])
}

// assertOnlyKeys checks that the map contains no keys outside the allowed set.
func assertOnlyKeys(t *testing.T, m map[string]interface{}, keys ...string) {
	t.Helper()
	for key := range m {
		assert.Contains(t, keys, key, "unexpected key %q in response", key)
	}
}

// assertListResponse extracts the data array from a paginated response.
func assertListResponse(t *testing.T, response map[string]interface{}) []map[string]interface{} {
	t.Helper()
	require.IsType(t, []interface{}{}, response["data"], "response.data should be an array")
	raw := response["data"].([]interface{})
	items := make([]map[string]interface{}, len(raw))
	for i, item := range raw {
		require.IsType(t, map[string]interface{}{}, item, "data[%d] should be an object", i)
		items[i] = item.(map[string]interface{})
	}
	return items
}

// assertPaginationLinks checks prev and next links in a paginated response.
// Pass nil for prev/next to assert they are absent, or a string to assert the exact value.
func assertPaginationLinks(t *testing.T, response map[string]interface{}, expectedPrev, expectedNext interface{}) {
	t.Helper()
	require.IsType(t, map[string]interface{}{}, response["links"])
	links := response["links"].(map[string]interface{})
	assert.Equal(t, expectedPrev, links["prev"])
	assert.Equal(t, expectedNext, links["next"])
}
