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

	_ "github.com/mattn/go-sqlite3"
	_ "github.com/lib/pq"
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
	query    string
	body     string
	headers  map[string][]string

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
	fixtures, err := testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Dialect(dbDriver),
		testfixtures.Directory("test/testdata/fixtures/"),
	)
	assert.Nil(t, err)

	err = fixtures.Load()
	assert.Nil(t, err)
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
				assert.Nil(t, err)

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

func TestApi(t *testing.T) {
	tests := []TestCase{
		{
			description: "non existing route",
			query:       "GET /v1/i-dont-exist",

			expectedCode:        404,
			expectedBody:        `{"title":"Not Found","detail":"Cannot GET /v1/i-dont-exist","status":404}`,
			expectedContentType: "application/problem+json",
		},
	}

	runTestCases(t, tests)
}

func TestPublishersEndpoints(t *testing.T) {
	tests := []TestCase{
		// GET /publishers
		{
			description:         "GET the first page on publishers",
			query:               "GET /v1/publishers",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 25, len(data))

				// Default pagination size is 25, so there's another page and
				// next cursor should be present
				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyIyMDE4LTExLTI3VDAwOjAwOjAwWiIsIjgxZmRhN2M0LTZiYmYtNDM4Ny04Zjg5LTI1OGMxZTZmYWZhMiJd", links["next"])

				assert.IsType(t, map[string]interface{}{}, data[0])
				firstPub := data[0].(map[string]interface{})
				assert.NotEmpty(t, firstPub["email"])

				assert.IsType(t, []interface{}{}, firstPub["codeHosting"])
				assert.Equal(t, 2, len(firstPub["codeHosting"].([]interface{})))

				match, err := regexp.MatchString(UUID_REGEXP, firstPub["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, firstPub["createdAt"].(string))
				assert.Nil(t, err)
				_, err = time.Parse(time.RFC3339, firstPub["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range firstPub {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "codeHosting", "email", "description", "active"}, key)
				}
			},
		},
		{
			description:         "GET all the publishers, except the non active ones",
			query:               "GET /v1/publishers?page[size]=100",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 27, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Nil(t, links["next"])

				assert.IsType(t, map[string]interface{}{}, data[0])
				firstPub := data[0].(map[string]interface{})
				assert.NotEmpty(t, firstPub["codeHosting"])

				assert.IsType(t, []interface{}{}, firstPub["codeHosting"])
				assert.Greater(t, len(firstPub["codeHosting"].([]interface{})), 0)

				match, err := regexp.MatchString(UUID_REGEXP, firstPub["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, firstPub["createdAt"].(string))
				assert.Nil(t, err)
				_, err = time.Parse(time.RFC3339, firstPub["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range firstPub {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "codeHosting", "email", "description", "active"}, key)
				}
			},
		},
		{
			description: "GET all publishers, including non active",
			query:       "GET /v1/publishers?all=true&page[size]=100",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 28, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Nil(t, links["next"])

				assert.IsType(t, map[string]interface{}{}, data[0])
				firstPub := data[0].(map[string]interface{})
				assert.NotEmpty(t, firstPub["codeHosting"])

				assert.IsType(t, []interface{}{}, firstPub["codeHosting"])
				assert.Greater(t, len(firstPub["codeHosting"].([]interface{})), 0)

				match, err := regexp.MatchString(UUID_REGEXP, firstPub["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, firstPub["createdAt"].(string))
				assert.Nil(t, err)
				_, err = time.Parse(time.RFC3339, firstPub["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range firstPub {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "codeHosting", "email", "description", "active"}, key)
				}
			},
		},
		{
			description: "GET with page[size] query param",
			query:       "GET /v1/publishers?page[size]=2",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 2, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyIyMDE4LTA1LTE2VDAwOjAwOjAwWiIsIjQ3ODA3ZTBjLTA2MTMtNGFlYS05OTE3LTU0NTVjYzZlZGRhZCJd", links["next"])
			},
		},
		// TODO
		// {
		// 	description: "GET with invalid format for page[size] query param",
		// 	query:    "GET /v1/publishers?page[size]=NOT_AN_INT",

		// 	expectedCode:        422,
		// 	expectedContentType: "application/json",
		// },
		// TODO
		// {
		// 	description: "GET with page[size] bigger than the max of 100",
		// 	query:    "GET /v1/publishers?page[size]=200",

		// 	expectedCode:        422,
		// 	expectedContentType: "application/json",
		// },
		{
			description: `GET with "page[after]" query param`,
			query:       "GET /v1/publishers?page[after]=WyIyMDE4LTExLTI3VDAwOjAwOjAwWiIsIjgxZmRhN2M0LTZiYmYtNDM4Ny04Zjg5LTI1OGMxZTZmYWZhMiJd",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 2, len(data))

				links := response["links"].(map[string]interface{})
				assert.Equal(t, "?page[before]=WyIyMDE4LTExLTI3VDAwOjAwOjAwWiIsIjkxZmRhN2M0LTZiYmYtNDM4Ny04Zjg5LTI1OGMxZTZmYWZhMiJd", links["prev"])
				assert.Nil(t, links["next"])
			},
		},
		{
			description: `GET with invalid "page[after]" query param`,
			query:       "GET /v1/publishers?page[after]=NOT_A_VALID_CURSOR",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Publishers`, response["title"])
				assert.Equal(t, "wrong cursor format in page[after] or page[before]", response["detail"])
			},
		},
		{
			description: "GET with page[before] query param",
			query:       "GET /v1/publishers?page[before]=WyIyMDE4LTExLTI3VDAwOjAwOjAwWiIsIjkxZmRhN2M0LTZiYmYtNDM4Ny04Zjg5LTI1OGMxZTZmYWZhMiJd",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 25, len(data))

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyIyMDE4LTExLTI3VDAwOjAwOjAwWiIsIjgxZmRhN2M0LTZiYmYtNDM4Ny04Zjg5LTI1OGMxZTZmYWZhMiJd", links["next"])
			},
		},
		{
			description: `GET with invalid "page[before]" query param`,
			query:       "GET /v1/publishers?page[before]=NOT_A_VALID_CURSOR",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Publishers`, response["title"])
				assert.Equal(t, "wrong cursor format in page[after] or page[before]", response["detail"])
			},
		},

		// GET /publishers/:id
		{
			description:         "Non-existent publisher",
			query:               "GET /v1/publishers/eea19c82-0449-11ed-bd84-d8bbc146d165",
			expectedCode:        404,
			expectedBody:        `{"title":"can't get Publisher","detail":"Publisher was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			query:               "GET /v1/publishers/15fda7c4-6bbf-4387-8f89-258c1e6fafb1",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.NotEmpty(t, response["codeHosting"])

				assert.IsType(t, []interface{}{}, response["codeHosting"])
				assert.Greater(t, len(response["codeHosting"].([]interface{})), 0)

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)
				_, err = time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range response {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "codeHosting", "email", "description", "active"}, key)
				}
			},
		},
		{
			description:         "GET publisher with alternativeId",
			query:               "GET /v1/publishers/alternative-id-12345",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "15fda7c4-6bbf-4387-8f89-258c1e6facb0", response["id"])
				assert.Equal(t, "alternative-id-12345", response["alternativeId"])

				_, err := time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)
				_, err = time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range response {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "codeHosting", "email", "description", "active", "alternativeId"}, key)
				}
			},
		},

		// POST /publishers
		{
			query: "POST /v1/publishers",
			body:  `{"description": "new description", "codeHosting": [{"url" : "https://www.example-testcase-1.com"}], "email":"example-testcase-1@example.com"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["codeHosting"])

				codeHosting := response["codeHosting"].([]interface{})
				assert.Equal(t, 1, len(codeHosting))

				firstCodeHosting := codeHosting[0].(map[string]interface{})
				assert.Equal(t, "https://example-testcase-1.com", firstCodeHosting["url"])
				assert.Equal(t, true, firstCodeHosting["group"])

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				_, err = time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				assert.Equal(t, true, response["active"])
				assert.Equal(t, "example-testcase-1@example.com", response["email"])

				// TODO: check the record was actually created in the database
				// TODO: check there are no dangling publishers_codeHosting
			},
		},
		{
			description: "POST publisher with alternativeId",
			query:       "POST /v1/publishers",
			body:        `{"alternativeId":"12345", "description":"new description", "codeHosting": [{"url" : "https://www.example-testcase-2.com"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["codeHosting"])
				assert.Equal(t, 1, len(response["codeHosting"].([]interface{})))

				// TODO: check codeHosting content
				assert.NotEmpty(t, response["codeHosting"])

				assert.Equal(t, "12345", response["alternativeId"])

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				_, err = time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)
			},
		},
		{
			description: "POST publisher with duplicate alternativeId",
			query: "POST /v1/publishers",
			body:  `{"alternativeId": "alternative-id-12345", "description":"new description", "codeHosting": [{"url" : "https://example-testcase-xx3.com"}], "email":"example-testcase-3-pass@example.com"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"description, alternativeId or codeHosting URL already exists","status":409}`,
		},
		{
			description: "POST publisher with alternativeId matching an existing id",
			query: "POST /v1/publishers",
			body:  `{"alternativeId": "2ded32eb-c45e-4167-9166-a44e18b8adde", "description":"new description", "codeHosting": [{"url" : "https://example-testcase-xx3.com"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"Publisher with id '2ded32eb-c45e-4167-9166-a44e18b8adde' already exists","status":409}`,
		},
		{
			description: "POST publisher with empty alternativeId",
			query: "POST /v1/publishers",
			body:  `{"alternativeId": "", "description":"new description", "codeHosting": [{"url" : "https://gitlab.example.com/repo"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"invalid format: alternativeId does not meet its size limits (too short)","status":422,"validationErrors":[{"field":"alternativeId","rule":"min","value":""}]}`,
		},
		{
			query: "POST /v1/publishers - NOT normalized URL validation passed",
			body:  `{"description":"new description", "codeHosting": [{"url" : "https://WwW.example-testcase-3.com"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["codeHosting"])
				assert.Equal(t, 1, len(response["codeHosting"].([]interface{})))

				// TODO: check codeHosting content
				assert.NotEmpty(t, response["codeHosting"])

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				_, err = time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)
			},
		},
		{
			description: "POST publishers with duplicate URL (when normalized)",
			query:       "POST /v1/publishers",
			body: `{"codeHosting": [{"url" : "https://1-a.exAMple.org/code/repo"}], "description":"new description"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"description, alternativeId or codeHosting URL already exists","status":409}`,
		},
		{
			description: "POST new publisher with an existing email",
			query:       "POST /v1/publishers",
			body:        `{"codeHosting": [{"url" : "https://new-url.example.com"}], "email":"foobar@1.example.org", "description": "new publisher description"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "foobar@1.example.org", response["email"])
			},
		},
		{
			description:    "POST new publisher with an existing email (not normalized)",
			query:          "POST /v1/publishers",
			body:     `{"codeHosting": [{"url" : "https://new-url.example.com"}], "email":"FoobaR@1.example.org", "description": "new publisher description"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "foobar@1.example.org", response["email"])
			},
		},
		{
			description:    "POST new publisher with no email",
			query:          "POST /v1/publishers",
			body: `{"codeHosting": [{"url" : "https://new-url.example.com"}], "description": "new publisher description"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "new publisher description", response["description"])

				email, exists := response["email"]
				assert.False(t, exists, "email key is present: %s", email)
			},
		},
		{
			description:    "POST new publisher with empty email",
			query:          "POST /v1/publishers",
			body: `{"email": "", "codeHosting": [{"url" : "https://new-url.example.com"}], "description": "new publisher description"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"invalid format: email is not a valid email","status":422,"validationErrors":[{"field":"email","rule":"email","value":""}]}`,
		},
		{
			query:    "POST /v1/publishers - Description already exist",
			body:     `{"codeHosting": [{"url" : "https://example-testcase-xx3.com"}], "description": "Publisher description 1"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"description, alternativeId or codeHosting URL already exists","status":409}`,
		},
		{
			description: "POST new publisher with no description",
			query:       "POST /v1/publishers",
			body:        `{"codeHosting": [{"url" : "https://WwW.example-testcase-3.com"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"invalid format: description is required","status":422,"validationErrors":[{"field":"description","rule":"required","value":""}]}`,
		},
		{
			description: "POST new publisher with empty description",
			query:       "POST /v1/publishers",
			body:        `{"description":"", "codeHosting": [{"url" : "https://WwW.example-testcase-3.com"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"invalid format: description is required","status":422,"validationErrors":[{"field":"description","rule":"required","value":""}]}`,
		},
		{
			description: "POST publisher with duplicate alternativeId",
			query: "POST /v1/publishers",
			body:  `{"alternativeId": "alternative-id-12345", "description":"new description", "codeHosting": [{"url" : "https://example-testcase-xx3.com"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"description, alternativeId or codeHosting URL already exists","status":409}`,
		},
		{
			description: "POST publishers with invalid payload",
			query:       "POST /v1/publishers",
			body:        `{"url": "-"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Publisher","detail":"invalid format: codeHosting is required, description is required","status":422,"validationErrors":[{"field":"codeHosting","rule":"required","value":""},{"field":"description","rule":"required","value":""}]}`,
		},
		{
			description: "POST publishers - wrong token",
			query:       "POST /v1/publishers",
			body:        `{"description":"new description", "codeHosting": [{"url" : "https://www.example-5.com"}]}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			query: "POST /v1/publishers with invalid JSON",
			body:  `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Publisher`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		{
			description: "POST publishers with optional boolean field set to false",
			query:       "POST /v1/publishers",
			body:        `{"active": false, "description": "new description", "codeHosting": [{"url" : "https://www.example.com"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, false, response["active"])
			},
		},
		{
			description: "POST publishers with codeHosting optional boolean field (group) set to false",
			query:       "POST /v1/publishers",
			body:        `{"description":"new description", "codeHosting": [{"url" : "https://www.example.com", "group": false}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["codeHosting"])

				codeHosting := response["codeHosting"].([]interface{})
				assert.Equal(t, 1, len(codeHosting))

				firstCodeHosting := codeHosting[0].(map[string]interface{})
				assert.Equal(t, "https://example.com", firstCodeHosting["url"])
				assert.Equal(t, false, firstCodeHosting["group"])
			},
		},
		{
			description: "POST publishers with validation errors",
			query:       "POST /v1/publishers",
			body:        `{"codeHosting": [{"url" : "a"}], "email":"b"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Publisher`, response["title"])
				assert.Equal(t, "invalid format: url is invalid, description is required, email is not a valid email", response["detail"])

				assert.IsType(t, []interface{}{}, response["validationErrors"])

				validationErrors := response["validationErrors"].([]interface{})
				assert.Equal(t, 3, len(validationErrors))

				firstValidationError := validationErrors[0].(map[string]interface{})

				for key := range firstValidationError {
					assert.Contains(t, []string{"field", "rule", "value"}, key)
				}
			},
		},
		{
			description: "POST publishers with empty body",
			query:       "POST /v1/publishers",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Publisher`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},

		// PATCH /publishers/:id
		{
			description: "PATCH non existing publisher",
			query:       "PATCH /v1/publishers/NO_SUCH_PUBLISHER",
			body:        ``,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedBody:        `{"title":"can't update Publisher","detail":"Publisher was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "PATCH a publisher",
			query: "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:  `{"description": "new PATCHed description", "codeHosting": [{"url": "https://gitlab.example.org/patched-repo"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "new PATCHed description", response["description"])
				assert.IsType(t, []interface{}{}, response["codeHosting"])

				codeHosting := response["codeHosting"].([]interface{})
				assert.Equal(t, 1, len(codeHosting))

				firstCodeHosting := codeHosting[0].(map[string]interface{})

				assert.Equal(t, "https://gitlab.example.org/patched-repo", firstCodeHosting["url"])
				assert.Equal(t, "2ded32eb-c45e-4167-9166-a44e18b8adde", response["id"])

				created, err := time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				updated, err := time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH publishers with no codeHosting (should leave current codeHosting untouched)",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `{"description": "new description"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "2ded32eb-c45e-4167-9166-a44e18b8adde", response["id"])
				assert.Equal(t, "new description", response["description"])
				assert.Equal(t, "foobar@1.example.org", response["email"])

				assert.IsType(t, []interface{}{}, response["codeHosting"])

				codeHosting := response["codeHosting"].([]interface{})
				assert.Equal(t, 2, len(codeHosting))

				firstCodeHosting := codeHosting[0].(map[string]interface{})
				assert.Equal(t, "https://1-a.example.org/code/repo", firstCodeHosting["url"])
				secondCodeHosting := codeHosting[1].(map[string]interface{})
				assert.Equal(t, "https://1-b.example.org/code/repo", secondCodeHosting["url"])

				assert.Equal(t, "2018-05-01T00:00:00Z", response["createdAt"])
				created, err := time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				updated, err := time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH publishers with empty codeHosting",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `{"description": "new description", "codeHosting": []}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody: `{"title":"can't update Publisher","detail":"invalid format: codeHosting does not meet its size limits (too few items)","status":422,"validationErrors":[{"field":"codeHosting","rule":"gt","value":""}]}`,
		},
		{
			description: "PATCH a publisher via alternativeId",
			query: "PATCH /v1/publishers/alternative-id-12345",
			body:  `{"description": "new PATCHed description via alternativeId", "codeHosting": [{"url": "https://gitlab.example.org/patched-repo"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "new PATCHed description via alternativeId", response["description"])
				assert.IsType(t, []interface{}{}, response["codeHosting"])

				codeHosting := response["codeHosting"].([]interface{})
				assert.Equal(t, 1, len(codeHosting))

				firstCodeHosting := codeHosting[0].(map[string]interface{})

				assert.Equal(t, "https://gitlab.example.org/patched-repo", firstCodeHosting["url"])
				assert.Equal(t, "15fda7c4-6bbf-4387-8f89-258c1e6facb0", response["id"])

				created, err := time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				updated, err := time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH a publisher with alternativeId matching an existing id",
			query: "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:  `{"alternativeId": "47807e0c-0613-4aea-9917-5455cc6eddad"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        409,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't update Publisher","detail":"Publisher with id '47807e0c-0613-4aea-9917-5455cc6eddad' already exists","status":409}`,
		},
		{
			description: "PATCH publishers - wrong token",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        ``,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "PATCH publisher with invalid JSON",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Publisher`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: make this pass
		// {
		// 	description: "PATCH publishers with JSON with extra fields",
		// 	query: "PATCH /v1/publishers",
		// 	body: `{"description": "new description", EXTRA_FIELD: "extra field not in schema"}`,
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 		"Content-Type":  {"application/json"},
		// 	},
		// 	expectedCode:        422,
		// 	expectedContentType: "application/problem+json",
		// 	validateFunc: func(t *testing.T, response map[string]interface{}) {
		// 		assert.Equal(t, `can't create Publisher`, response["title"])
		// 		assert.Equal(t, "invalid json", response["detail"])
		// 	},
		// },
		{
			description: "PATCH publisher with validation errors",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        `{"description": "new description", "codeHosting": [{"url": "INVALID_URL"}]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody: `{"title":"can't update Publisher","detail":"invalid format: url is invalid","status":422,"validationErrors":[{"field":"url","rule":"url","value":"INVALID_URL"}]}`,
		},
		{
			description: "PATCH publishers with empty body",
			query:       "PATCH /v1/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			expectedBody: `{"title":"can't update Publisher","detail":"invalid json","status":400}`,
		},
		// TODO: enforce this?
		// {
		// 	query: "PATCH /v1/publishers with no Content-Type",
		// 	body:  "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// }

		// DELETE /publishers/:id
		{
			description: "Delete non-existent publishers",
			query:       "DELETE /v1/publishers/eea19c82-0449-11ed-bd84-d8bbc146d165",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedBody:        `{"title":"can't delete Publisher","detail":"Publisher was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "DELETE publishers with bad authentication",
			query:       "DELETE /v1/publishers/15fda7c4-6bbf-4387-8f89-258c1e6fafb1",
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			query:    "DELETE /v1/publishers/15fda7c4-6bbf-4387-8f89-258c1e6fafb1",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        204,
			expectedBody:        "",
			expectedContentType: "",
		},
		{
			description: "DELETE publisher via alternativeId",
			query: "DELETE /v1/publishers/alternative-id-12345",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        204,
			expectedBody:        "",
			expectedContentType: "",
		},

		// WebHooks

		// GET /publishers/:id/webhooks
		{
			query:    "GET /v1/publishers/47807e0c-0613-4aea-9917-5455cc6eddad/webhooks",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 1, len(data))

				// Default pagination size is 25, so all this publishers's logs fit into a page
				// and cursors should be empty
				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Nil(t, links["next"])

				assert.IsType(t, map[string]interface{}{}, data[0])
				firstWebhook := data[0].(map[string]interface{})
				assert.Equal(t, "https://6-b.example.org/receiver", firstWebhook["url"])
				assert.Equal(t, "1702cd06-fffb-4d20-8f55-73e2a00ee052", firstWebhook["id"])
				assert.Equal(t, "2018-07-15T00:00:00Z", firstWebhook["createdAt"])
				assert.Equal(t, "2018-07-15T00:00:00Z", firstWebhook["updatedAt"])

				for key := range firstWebhook {
					assert.Contains(t, []string{"id", "url", "createdAt", "updatedAt"}, key)
				}
			},
		},
		{
			description: "GET webhooks for non existing publisher",
			query:       "GET /v1/publishers/NO_SUCH_publishers/webhooks",

			expectedCode:        404,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't find resource`, response["title"])
				assert.Equal(t, "resource was not found", response["detail"])
			},
		},
		{
			description: "GET webhooks for publisher without webhooks",
			query:       "GET /v1/publishers/b97446f8-fe06-472c-9b26-c40150cac77f/webhooks",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 0, len(data))
			},
		},
		{
			description: "GET with page[size] query param",
			query:       "GET /v1/publishers/d6ddc11a-ff85-4f0f-bb87-df38b2a9b394/webhooks?page[size]=1",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 1, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyIyMDE4LTA3LTE1VDAwOjAwOjAwWiIsIjhmMzczYThjLTFmNTUtNDVlNC04NTQ5LTA1Y2Q2MzJhMmFkZCJd", links["next"])
			},
		},

		// POST /publishers/:id/webhooks
		{
			description: "POST webhook for non existing publisher",
			query:       "POST /v1/publishers/NO_SUCH_publishers/webhooks",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        404,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't find resource`, response["title"])
				assert.Equal(t, "resource was not found", response["detail"])
			},
		},
		{
			query:    "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks",
			body:     `{"url": "https://new.example.org", "secret": "xyz"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://new.example.org", response["url"])

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				_, err = time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range response {
					assert.Contains(t, []string{"id", "url", "createdAt", "updatedAt"}, key)
				}

				// TODO: check the record was actually created in the database
			},
		},
		{
			description: "POST publishers webhook - wrong token",
			query:       "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks",
			body:        `{"url": "https://new.example.org"}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "POST webhook with invalid JSON",
			query:       "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks",
			body:        `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Webhook`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: make this pass
		// {
		// 	description: "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks with JSON with extra fields",
		// 	body: `{"url": "https://new.example.org", EXTRA_FIELD: "extra field not in schema"}`,
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 		"Content-Type":  {"application/json"},
		// 	},
		// 	expectedCode:        422,
		// 	expectedContentType: "application/problem+json",
		// 	validateFunc: func(t *testing.T, response map[string]interface{}) {
		// 		assert.Equal(t, `can't create Webhook`, response["title"])
		// 		assert.Equal(t, "invalid json", response["detail"])
		// 	},
		// },
		{
			description: "POST webhook with validation errors",
			query:       "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks",
			body:        `{"url": "INVALID_URL"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Webhook`, response["title"])
				assert.Equal(t, "invalid format: url is invalid", response["detail"])

				assert.IsType(t, []interface{}{}, response["validationErrors"])

				validationErrors := response["validationErrors"].([]interface{})
				assert.Equal(t, 1, len(validationErrors))

				firstValidationError := validationErrors[0].(map[string]interface{})

				for key := range firstValidationError {
					assert.Contains(t, []string{"field", "rule", "value"}, key)
				}
			},
		},
		{
			description: "POST webhook with empty body",
			query:       "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Webhook`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: enforce this?
		// {
		// 	query: "POST /v1/publishers/98a069f7-57b0-464d-b300-4b4b336297a0/webhooks with no Content-Type",
		// 	body:  "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// },
	}

	runTestCases(t, tests)
}

func TestSoftwareEndpoints(t *testing.T) {
	tests := []TestCase{
		// GET /software
		{
			description: "GET the first page on software",
			query:       "GET /v1/software",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 25, len(data))

				// Default pagination size is 25, so there's another page and
				// next cursor should be present
				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyIyMDE1LTA0LTI2VDAwOjAwOjAwWiIsIjEyNDI4MGQ3LTc1NTItNGZmZS05MzlmLWY0NjY5N2NjMGU4YSJd", links["next"])

				assert.IsType(t, map[string]interface{}{}, data[0])
				firstSoftware := data[0].(map[string]interface{})
				assert.NotEmpty(t, firstSoftware["publiccodeYml"])

				assert.Equal(t, "https://1-a.example.org/code/repo", firstSoftware["url"])

				assert.IsType(t, []interface{}{}, firstSoftware["aliases"])
				assert.Equal(t, 1, len(firstSoftware["aliases"].([]interface{})))

				match, err := regexp.MatchString(UUID_REGEXP, firstSoftware["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, firstSoftware["createdAt"].(string))
				assert.Nil(t, err)
				_, err = time.Parse(time.RFC3339, firstSoftware["updatedAt"].(string))
				assert.Nil(t, err)

				assert.Equal(t, true, firstSoftware["active"])

				for key := range firstSoftware {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active"}, key)
				}
			},
		},
		{
			description: "GET all the software, except the non active ones",
			query:       "GET /v1/software?page[size]=100",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 30, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Nil(t, links["next"])

				assert.IsType(t, map[string]interface{}{}, data[0])
				firstSoftware := data[0].(map[string]interface{})
				assert.NotEmpty(t, firstSoftware["publiccodeYml"])

				assert.Equal(t, "https://1-a.example.org/code/repo", firstSoftware["url"])

				assert.IsType(t, []interface{}{}, firstSoftware["aliases"])
				assert.Equal(t, 1, len(firstSoftware["aliases"].([]interface{})))

				match, err := regexp.MatchString(UUID_REGEXP, firstSoftware["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				assert.Equal(t, true, firstSoftware["active"])

				_, err = time.Parse(time.RFC3339, firstSoftware["createdAt"].(string))
				assert.Nil(t, err)
				_, err = time.Parse(time.RFC3339, firstSoftware["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range firstSoftware {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active"}, key)
				}
			},
		},
		{
			description: "GET all software, including non active",
			query:       "GET /v1/software?all=true&page[size]=100",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 31, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Nil(t, links["next"])

				assert.IsType(t, map[string]interface{}{}, data[0])
				firstSoftware := data[0].(map[string]interface{})
				assert.NotEmpty(t, firstSoftware["publiccodeYml"])

				assert.Equal(t, "https://1-a.example.org/code/repo", firstSoftware["url"])

				assert.IsType(t, []interface{}{}, firstSoftware["aliases"])
				assert.Equal(t, 1, len(firstSoftware["aliases"].([]interface{})))

				match, err := regexp.MatchString(UUID_REGEXP, firstSoftware["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, firstSoftware["createdAt"].(string))
				assert.Nil(t, err)
				_, err = time.Parse(time.RFC3339, firstSoftware["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range firstSoftware {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active"}, key)
				}
			},
		},
		{
			description: "GET software with a specific URL",
			query:       "GET /v1/software?url=https://1-a.example.org/code/repo",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 1, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Nil(t, links["next"])

				assert.IsType(t, map[string]interface{}{}, data[0])
				firstSoftware := data[0].(map[string]interface{})
				assert.Equal(t, "-", firstSoftware["publiccodeYml"])

				assert.Equal(t, "https://1-a.example.org/code/repo", firstSoftware["url"])

				assert.IsType(t, []interface{}{}, firstSoftware["aliases"])
				assert.Equal(t, 1, len(firstSoftware["aliases"].([]interface{})))

				assert.Equal(t, "c353756e-8597-4e46-a99b-7da2e141603b", firstSoftware["id"])

				assert.Equal(t, "2014-05-01T00:00:00Z", firstSoftware["createdAt"])
				assert.Equal(t, "2014-05-01T00:00:00Z", firstSoftware["updatedAt"])

				for key := range firstSoftware {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active"}, key)
				}
			},
		},
		{
			description: "GET software with a specific URL that doesn't exist",
			query:       "GET /v1/software?url=https://no.such.url.in.db.example.org/code/repo",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 0, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Nil(t, links["next"])
			},
		},
		{
			description: "GET with page[size] query param",
			query:       "GET /v1/software?page[size]=2",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 2, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyIyMDE0LTA1LTE2VDAwOjAwOjAwWiIsIjlmMTM1MjY4LWEzN2UtNGVhZC05NmVjLWU0YTI0YmI5MzQ0YSJd", links["next"])
			},
		},
		// TODO
		// {
		// 	description: "GET with invalid format for page[size] query param",
		// 	query:    "GET /v1/software?page[size]=NOT_AN_INT",

		// 	expectedCode:        422,
		// 	expectedContentType: "application/json",
		// },
		// TODO
		// {
		// 	description: "GET with page[size] bigger than the max of 100",
		// 	query:    "GET /v1/software?page[size]=200",

		// 	expectedCode:        422,
		// 	expectedContentType: "application/json",
		// },
		{
			description: `GET with "page[after]" query param`,
			query:       "GET /v1/software?page[after]=WyIyMDE1LTA0LTI2VDAwOjAwOjAwWiIsIjEyNDI4MGQ3LTc1NTItNGZmZS05MzlmLWY0NjY5N2NjMGU4YSJd",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 5, len(data))

				links := response["links"].(map[string]interface{})
				assert.Equal(t, "?page[before]=WyIyMDE1LTA1LTExVDAwOjAwOjAwWiIsIjgzZTdhMzVlLTMyOGItNDg5MS1iNjBiLTU5NzkyZTAxYzU5ZSJd", links["prev"])
				assert.Nil(t, links["next"])
			},
		},
		{
			description: `GET with invalid "page[after]" query param`,
			query:       "GET /v1/software?page[after]=NOT_A_VALID_CURSOR",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Software`, response["title"])
				assert.Equal(t, "wrong cursor format in page[after] or page[before]", response["detail"])
			},
		},
		{
			description: "GET with page[before] query param",
			query:       "GET /v1/software?page[before]=WyIyMDE1LTA1LTExVDAwOjAwOjAwWiIsIjgzZTdhMzVlLTMyOGItNDg5MS1iNjBiLTU5NzkyZTAxYzU5ZSJd",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 25, len(data))

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyIyMDE1LTA0LTI2VDAwOjAwOjAwWiIsIjEyNDI4MGQ3LTc1NTItNGZmZS05MzlmLWY0NjY5N2NjMGU4YSJd", links["next"])
			},
		},
		{
			description: `GET with invalid "page[before]" query param`,
			query:       "GET /v1/software?page[before]=NOT_A_VALID_CURSOR",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Software`, response["title"])
				assert.Equal(t, "wrong cursor format in page[after] or page[before]", response["detail"])
			},
		},
		{
			description: `GET with "from" query param`,
			query:       "GET /v1/software?from=2015-04-01T09:56:23Z",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 7, len(data))
			},
		},
		{
			description: `GET with invalid "from" query param`,
			query:       "GET /v1/software?from=3",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Software`, response["title"])
				assert.Equal(t, "invalid date time format (RFC 3339 needed)", response["detail"])
			},
		},
		{
			description: `GET with "to" query param`,
			query:       "GET /v1/software?to=2014-11-01T09:56:23Z",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 13, len(data))
			},
		},
		{
			description: `GET with invalid "to" query param`,
			query:       "GET /v1/software?to=3",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Software`, response["title"])
				assert.Equal(t, "invalid date time format (RFC 3339 needed)", response["detail"])
			},
		},

		// GET /software/:id
		{
			description:         "Non-existent software",
			query:               "GET /v1/software/eea19c82-0449-11ed-bd84-d8bbc146d165",
			expectedCode:        404,
			expectedBody:        `{"title":"can't get Software","detail":"Software was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			query:               "GET /v1/software/e7576e7f-9dcf-4979-b9e9-d8cdcad3b60e",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.NotEmpty(t, response["publiccodeYml"])

				assert.Equal(t, "https://8-a.example.org/code/repo", response["url"])

				assert.IsType(t, []interface{}{}, response["aliases"])
				assert.Equal(t, 1, len(response["aliases"].([]interface{})))

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)
				_, err = time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range response {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active"}, key)
				}
			},
		},

		// POST /software
		{
			query: "POST /v1/software",
			body:  `{"publiccodeYml": "-", "url": "https://software.example.org"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://software.example.org", response["url"])
				assert.NotEmpty(t, response["publiccodeYml"])

				assert.IsType(t, []interface{}{}, response["aliases"])
				assert.Empty(t, response["aliases"].([]interface{}))

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				_, err = time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				assert.Equal(t, true, response["active"])

				for key := range response {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active"}, key)
				}

				// TODO: check the record was actually created in the database
				// TODO: check there are no dangling software_urls
			},
		},
		{
			description: "POST software with aliases",
			query:       "POST /v1/software",
			body:        `{"publiccodeYml": "-", "url": "https://software.example.org", "aliases": ["https://software-1.example.org", "https://software-2.example.org"]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://software.example.org", response["url"])
				assert.NotEmpty(t, response["publiccodeYml"])

				assert.IsType(t, []interface{}{}, response["aliases"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 2, len(aliases))

				assert.Equal(t, "https://software-1.example.org", aliases[0])
				assert.Equal(t, "https://software-2.example.org", aliases[1])

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				_, err = time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range response {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "url", "aliases", "publiccodeYml", "active"}, key)
				}

				// TODO: check the record was actually created in the database
				// TODO: check there are no dangling software_urls
			},
		},

		{
			description: "POST software with invalid payload",
			query:       "POST /v1/software",
			body:        `{"publiccodeYml": "-"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Software","detail":"invalid format: url is required","status":422,"validationErrors":[{"field":"url","rule":"required","value":""}]}`,
		},
		{
			description: "POST software - wrong token",
			query:       "POST /v1/software",
			body:        `{"publiccodeYml":  "-", "url": "https://software.example.org"}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			query: "POST /v1/software with invalid JSON",
			body:  `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Software`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: make this pass
		// {
		// 	descrption: "POST /v1/software with JSON with extra fields",
		// 	query: "POST /v1/software",
		// 	body: `{"publiccodeYml": "-", EXTRA_FIELD: "extra field not in schema"}`,
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 		"Content-Type":  {"application/json"},
		// 	},
		// 	expectedCode:        422,
		// 	expectedContentType: "application/problem+json",
		// 	validateFunc: func(t *testing.T, response map[string]interface{}) {
		// 		assert.Equal(t, `can't create Software`, response["title"])
		// 		assert.Equal(t, "invalid json", response["detail"])
		// 	},
		// },
		{
			description: "POST software with optional boolean field set to false",
			query:       "POST /v1/software",
			body:        `{"active": false, "url": "https://example.org", "publiccodeYml": "-"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, false, response["active"])
			},
		},
		{
			description: "POST software with validation errors",
			query:       "POST /v1/software",
			body:        `{"url":"", "publiccodeYml": "-"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Software`, response["title"])
				assert.Equal(t, "invalid format: url is required", response["detail"])

				assert.IsType(t, []interface{}{}, response["validationErrors"])

				validationErrors := response["validationErrors"].([]interface{})
				assert.Equal(t, 1, len(validationErrors))

				firstValidationError := validationErrors[0].(map[string]interface{})

				for key := range firstValidationError {
					assert.Contains(t, []string{"field", "rule", "value"}, key)
				}
			},
		},
		{
			description: "POST software with empty body",
			query:       "POST /v1/software",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Software`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: enforce this?
		// {
		// 	query: "POST /v1/software with no Content-Type",
		// 	body:  "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// }

		// PATCH /software/:id
		{
			description: "PATCH non-existing software",
			query:       "PATCH /v1/software/NO_SUCH_SOFTWARE",
			body:        ``,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedBody:        `{"title":"can't update Software","detail":"Software was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			query: "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:  `{"publiccodeYml": "publiccodedata", "url": "https://software-new.example.org", "aliases": ["https://software.example.com", "https://software-old.example.org"]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://software-new.example.org", response["url"])

				assert.IsType(t, []interface{}{}, response["aliases"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 2, len(aliases))

				assert.Equal(t, "https://software-old.example.org", aliases[0])
				assert.Equal(t, "https://software.example.com", aliases[1])

				assert.Equal(t, "publiccodedata", response["publiccodeYml"])

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				created, err := time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				updated, err := time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH software with no aliases (should leave current aliases untouched)",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"publiccodeYml": "publiccodedata", "url": "https://software-new.example.org"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://software-new.example.org", response["url"])

				assert.IsType(t, []interface{}{}, response["aliases"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 2, len(aliases))

				assert.Equal(t, "https://18-a.example.org/code/repo", aliases[0])
				assert.Equal(t, "https://18-b.example.org/code/repo", aliases[1])

				assert.Equal(t, "publiccodedata", response["publiccodeYml"])

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				created, err := time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				updated, err := time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH software with empty aliases (should remove aliases)",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"publiccodeYml": "publiccodedata", "url": "https://software-new.example.org", "aliases": []}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://software-new.example.org", response["url"])

				assert.IsType(t, []interface{}{}, response["aliases"])

				aliases := response["aliases"].([]interface{})
				assert.Equal(t, 0, len(aliases))

				assert.Equal(t, "publiccodedata", response["publiccodeYml"])

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				created, err := time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				updated, err := time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				assert.Greater(t, updated, created)
			},
		},
		{
			description: "PATCH software - wrong token",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        ``,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "PATCH software with invalid JSON",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Software`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: make this pass
		// {
		// 	description: "PATCH software with JSON with extra fields",
		// 	query: "PATCH /v1/software",
		// 	body: `{"publiccodeYml": "-", EXTRA_FIELD: "extra field not in schema"}`,
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 		"Content-Type":  {"application/json"},
		// 	},
		// 	expectedCode:        422,
		// 	expectedContentType: "application/problem+json",
		// 	validateFunc: func(t *testing.T, response map[string]interface{}) {
		// 		assert.Equal(t, `can't create Software`, response["title"])
		// 		assert.Equal(t, "invalid json", response["detail"])
		// 	},
		// },
		{
			description: "PATCH software with validation errors",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        `{"url": "INVALID_URL", "publiccodeYml": "-"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Software`, response["title"])
				assert.Equal(t, "invalid format: url is invalid", response["detail"])

				assert.IsType(t, []interface{}{}, response["validationErrors"])

				validationErrors := response["validationErrors"].([]interface{})
				assert.Equal(t, 1, len(validationErrors))

				firstValidationError := validationErrors[0].(map[string]interface{})

				for key := range firstValidationError {
					assert.Contains(t, []string{"field", "rule", "value"}, key)
				}
			},
		},
		{
			description: "PATCH software with empty body",
			query:       "PATCH /v1/software/59803fb7-8eec-4fe5-a354-8926009c364a",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Software`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: enforce this?
		// {
		// 	query: "POST /v1/software with no Content-Type",
		// 	body:  "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// }

		// DELETE /software/:id
		{
			description:         "Delete non-existent software",
			query:               "DELETE /v1/software/eea19c82-0449-11ed-bd84-d8bbc146d165",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			expectedBody:        `{"title":"can't delete Software","detail":"Software was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "DELETE software with bad authentication",
			query:       "DELETE /v1/software/11e101c4-f989-4cc4-a665-63f9f34e83f6",
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			query:    "DELETE /v1/software/11e101c4-f989-4cc4-a665-63f9f34e83f6",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        204,
			expectedBody:        "",
			expectedContentType: "",
		},
		// TODO: check there are no dangling software_urls

		// GET /software/:id/logs
		{
			query:    "GET /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 3, len(data))

				// Default pagination size is 25, so all this software's logs fit into a page
				// and cursors should be empty
				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Nil(t, links["next"])

				var prevCreatedAt *time.Time = nil
				for _, l := range data {
					assert.IsType(t, map[string]interface{}{}, l)
					log := l.(map[string]interface{})

					assert.NotEmpty(t, log["message"])

					match, err := regexp.MatchString(UUID_REGEXP, log["id"].(string))
					assert.Nil(t, err)
					assert.True(t, match)

					createdAt, err := time.Parse(time.RFC3339, log["createdAt"].(string))
					assert.Nil(t, err)

					_, err = time.Parse(time.RFC3339, log["updatedAt"].(string))
					assert.Nil(t, err)

					assert.Equal(t, "/software/c353756e-8597-4e46-a99b-7da2e141603b", log["entity"])

					for key := range log {
						assert.Contains(t, []string{"id", "createdAt", "updatedAt", "message", "entity"}, key)
					}

					// Check the logs are ordered by descending createdAt
					if prevCreatedAt != nil {
						assert.GreaterOrEqual(t, *prevCreatedAt, createdAt)
					}

					prevCreatedAt = &createdAt
				}
			},
		},
		{
			description: "GET logs for non existing software",
			query:       "GET /v1/software/NO_SUCH_SOFTWARE/logs",

			expectedCode:        404,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Software`, response["title"])
				assert.Equal(t, "Software was not found", response["detail"])
			},
		},
		{
			description: "GET with page[size] query param",
			query:       "GET /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs?page[size]=2",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 2, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyIyMDEwLTAxLTE1VDIzOjU5OjU5WiIsIjEyZjMwZDllLTA0MmUtMTFlZC04ZGRjLWQ4YmJjMTQ2ZDE2NSJd", links["next"])
			},
		},

		// POST /software/:id/logs
		{
			description: "POST logs for non existing software",
			query:       "POST /v1/software/NO_SUCH_SOFTWARE/logs",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        404,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Log`, response["title"])
				assert.Equal(t, "Software was not found", response["detail"])
			},
		},
		{
			query:    "POST /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs",
			body:     `{"message": "New software log from test suite"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "New software log from test suite", response["message"])

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				_, err = time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				assert.Equal(t, "/software/c353756e-8597-4e46-a99b-7da2e141603b", response["entity"])

				// TODO: check the record was actually created in the database
			},
		},
		{
			description: "POST software log - wrong token",
			query:       "POST /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs",
			body:        `{"message": "new log"}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "POST log with invalid JSON",
			query:       "POST /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs",
			body:        `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Log`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: make this pass
		// {
		// 	description: "POST /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs with JSON with extra fields",
		// 	body: `{"message": "new log", EXTRA_FIELD: "extra field not in schema"}`,
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 		"Content-Type":  {"application/json"},
		// 	},
		// 	expectedCode:        422,
		// 	expectedContentType: "application/problem+json",
		// 	validateFunc: func(t *testing.T, response map[string]interface{}) {
		// 		assert.Equal(t, `can't create Log`, response["title"])
		// 		assert.Equal(t, "invalid json", response["detail"])
		// 	},
		// },
		{
			description: "POST log with validation errors",
			query:       "POST /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs",
			body:        `{"message": ""}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Log`, response["title"])
				assert.Equal(t, "invalid format: message is required", response["detail"])

				assert.IsType(t, []interface{}{}, response["validationErrors"])

				validationErrors := response["validationErrors"].([]interface{})
				assert.Equal(t, 1, len(validationErrors))

				firstValidationError := validationErrors[0].(map[string]interface{})

				for key := range firstValidationError {
					assert.Contains(t, []string{"field", "rule", "value"}, key)
				}
			},
		},
		{
			description: "POST log with empty body",
			query:       "POST /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Log`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: enforce this?
		// {
		// 	query: "POST /v1/logs with no Content-Type",
		// 	body:  "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// },

		// GET /software/:id/webhooks
		{
			query:    "GET /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 1, len(data))

				// Default pagination size is 25, so all this software's logs fit into a page
				// and cursors should be empty
				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Nil(t, links["next"])

				assert.IsType(t, map[string]interface{}{}, data[0])
				firstWebhook := data[0].(map[string]interface{})
				assert.Equal(t, "https://1-b.example.org/receiver", firstWebhook["url"])
				assert.Equal(t, "007bc84a-7e2d-43a0-b7e1-a256d4114aa7", firstWebhook["id"])
				assert.Equal(t, "2017-05-01T00:00:00Z", firstWebhook["createdAt"])
				assert.Equal(t, "2017-05-01T00:00:00Z", firstWebhook["updatedAt"])

				for key := range firstWebhook {
					assert.Contains(t, []string{"id", "url", "createdAt", "updatedAt"}, key)
				}
			},
		},
		{
			description: "GET webhooks for non existing software",
			query:       "GET /v1/software/NO_SUCH_SOFTWARE/webhooks",

			expectedCode:        404,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't find resource`, response["title"])
				assert.Equal(t, "resource was not found", response["detail"])
			},
		},
		{
			description: "GET webhooks for software without webhooks",
			query:       "GET /v1/software/e7576e7f-9dcf-4979-b9e9-d8cdcad3b60e/webhooks",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 0, len(data))
			},
		},
		{
			description: "GET with page[size] query param",
			query:       "GET /v1/software/9f135268-a37e-4ead-96ec-e4a24bb9344a/webhooks?page[size]=1",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 1, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyIyMDE3LTA1LTAxVDAwOjAwOjAwWiIsImU3ZjZkYmRhLWMzZjUtNGIyZi1iM2Q4LTM5YTM0MDI2ZTYwYSJd", links["next"])
			},
		},

		// POST /software/:id/webhooks
		{
			description: "POST webhooks for non existing software",
			query:       "POST /v1/software/NO_SUCH_SOFTWARE/webhooks",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},

			expectedCode:        404,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't find resource`, response["title"])
				assert.Equal(t, "resource was not found", response["detail"])
			},
		},
		{
			query:    "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks",
			body:     `{"url": "https://new.example.org", "secret": "xyz"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "https://new.example.org", response["url"])

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				_, err = time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range response {
					assert.Contains(t, []string{"id", "url", "createdAt", "updatedAt"}, key)
				}

				// TODO: check the record was actually created in the database
			},
		},
		{
			description: "POST software webhook - wrong token",
			query:       "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks",
			body:        `{"url": "https://new.example.org"}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "POST webhook with invalid JSON",
			query:       "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks",
			body:        `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Webhook`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: make this pass
		// {
		// 	description: "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks with JSON with extra fields",
		// 	body: `{"url": "https://new.example.org", EXTRA_FIELD: "extra field not in schema"}`,
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 		"Content-Type":  {"application/json"},
		// 	},
		// 	expectedCode:        422,
		// 	expectedContentType: "application/problem+json",
		// 	validateFunc: func(t *testing.T, response map[string]interface{}) {
		// 		assert.Equal(t, `can't create Webhook`, response["title"])
		// 		assert.Equal(t, "invalid json", response["detail"])
		// 	},
		// },
		{
			description: "POST webhook with validation errors",
			query:       "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks",
			body:        `{"url": "INVALID_URL"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Webhook`, response["title"])
				assert.Equal(t, "invalid format: url is invalid", response["detail"])

				assert.IsType(t, []interface{}{}, response["validationErrors"])

				validationErrors := response["validationErrors"].([]interface{})
				assert.Equal(t, 1, len(validationErrors))

				firstValidationError := validationErrors[0].(map[string]interface{})

				for key := range firstValidationError {
					assert.Contains(t, []string{"field", "rule", "value"}, key)
				}
			},
		},
		{
			description: "POST webhook with empty body",
			query:       "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Webhook`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: enforce this?
		// {
		// 	query: "POST /v1/software/c5dec6fa-8a01-4881-9e7d-132770d4214d/webhooks with no Content-Type",
		// 	body:  "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// },
	}

	runTestCases(t, tests)
}

func TestLogsEndpoints(t *testing.T) {
	tests := []TestCase{
		// GET /logs
		{
			query:               "GET /v1/logs",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 21, len(data))

				// Default pagination size is 25, so all the logs fit into a page
				// and cursors should be empty
				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Nil(t, links["next"])

				for _, l := range data {
					assert.IsType(t, map[string]interface{}{}, l)
					log := l.(map[string]interface{})
					assert.NotEmpty(t, log["message"])

					match, err := regexp.MatchString(UUID_REGEXP, log["id"].(string))
					assert.Nil(t, err)
					assert.True(t, match)

					createdAt, err := time.Parse(time.RFC3339, log["createdAt"].(string))
					assert.Nil(t, err)
					_, err = time.Parse(time.RFC3339, log["updatedAt"].(string))
					assert.Nil(t, err)

					// Only certain logs from the fixtures have an associated entity.
					//
					// FIXME: This is ugly, see the issue about improving tests:
					// https://github.com/italia/developers-italia-api/issues/91
					if log["id"] == "2dfb2bc2-042d-11ed-9338-d8bbc146d165" ||
						log["id"] == "12f30d9e-042e-11ed-8ddc-d8bbc146d165" ||
						log["id"] == "18a70362-042e-11ed-b793-d8bbc146d165" {
						assert.Equal(t, "/software/c353756e-8597-4e46-a99b-7da2e141603b", log["entity"])
					} else if log["id"] == "53650508-042e-11ed-9b84-d8bbc146d165" {
						assert.Equal(t, "/publishers/2ded32eb-c45e-4167-9166-a44e18b8adde", log["entity"])
					} else {
						assert.Nil(t, log["entity"])
					}

					var prevCreatedAt *time.Time = nil
					for key := range log {
						assert.Contains(t, []string{"id", "createdAt", "updatedAt", "message", "entity"}, key)
					}

					// Check the logs are ordered by descending createdAt
					if prevCreatedAt != nil {
						assert.GreaterOrEqual(t, *prevCreatedAt, createdAt)
					}

					prevCreatedAt = &createdAt
				}
			},
		},
		{
			description: "GET with page[size] query param",
			query:       "GET /v1/logs?page[size]=3",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 3, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyIyMDEwLTA3LTAxVDIzOjU5OjU5WiIsIjg1MWZlMGY0LTA0MmUtMTFlZC05MzNlLWQ4YmJjMTQ2ZDE2NSJd", links["next"])
			},
		},
		// TODO
		// {
		// 	description: "GET with invalid format for page[size] query param",
		// 	query:    "GET /v1/logs?page[size]=NOT_AN_INT",
		// 	expectedCode:        422,
		// 	expectedContentType: "application/json",
		// },
		{
			description: `GET with "page[after]" query param`,
			query:       "GET /v1/logs?page[after]=WyIyMDEwLTA3LTAxVDIzOjU5OjU5WiIsIjg1MWZlMGY0LTA0MmUtMTFlZC05MzNlLWQ4YmJjMTQ2ZDE2NSJd",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 18, len(data))

				links := response["links"].(map[string]interface{})
				assert.Equal(t, "?page[before]=WyIyMDEwLTA2LTMwVDIzOjU5OjU5WiIsIjgyNTZmODgwLTA0MmUtMTFlZC04MmI5LWQ4YmJjMTQ2ZDE2NSJd", links["prev"])
				assert.Nil(t, links["next"])
			},
		},
		{
			description: `GET with invalid "page[after]" query param`,
			query:       "GET /v1/logs?page[after]=NOT_A_VALID_CURSOR",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Logs`, response["title"])
				assert.Equal(t, "wrong cursor format in page[after] or page[before]", response["detail"])
			},
		},
		{
			description: "GET with page[before] query param",
			query:       "GET /v1/logs?page[before]=WyIyMDEwLTA2LTMwVDIzOjU5OjU5WiIsIjgyNTZmODgwLTA0MmUtMTFlZC04MmI5LWQ4YmJjMTQ2ZDE2NSJd",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 3, len(data))

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyIyMDEwLTA3LTAxVDIzOjU5OjU5WiIsIjg1MWZlMGY0LTA0MmUtMTFlZC05MzNlLWQ4YmJjMTQ2ZDE2NSJd", links["next"])
			},
		},
		{
			description: `GET with invalid "page[before]" query param`,
			query:       "GET /v1/logs?page[before]=NOT_A_VALID_CURSOR",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Logs`, response["title"])
				assert.Equal(t, "wrong cursor format in page[after] or page[before]", response["detail"])
			},
		},
		{
			description: `GET with "from" query param`,
			query:       "GET /v1/logs?from=2010-03-01T09:56:23Z",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 15, len(data))
			},
		},
		{
			description: `GET with invalid "from" query param`,
			query:       "GET /v1/logs?from=3",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Logs`, response["title"])
				assert.Equal(t, "invalid date time format (RFC 3339 needed)", response["detail"])
			},
		},
		{
			description: `GET with "to" query param`,
			query:       "GET /v1/logs?to=2010-03-01T09:56:23Z",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 6, len(data))
			},
		},
		{
			description: `GET with invalid "to" query param`,
			query:       "GET /v1/logs?to=3",

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Logs`, response["title"])
				assert.Equal(t, "invalid date time format (RFC 3339 needed)", response["detail"])
			},
		},
		{
			description:         "Non-existent log",
			query:               "GET /v1/logs/eea19c82-0449-11ed-bd84-d8bbc146d165",
			expectedCode:        404,
			expectedBody:        `{"title":"can't get Log","detail":"Log was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		// POST /logs
		{
			description: "POST with valid body",
			query:       "POST /v1/logs",
			body:        `{"message": "New log from test suite"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "New log from test suite", response["message"])

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				_, err = time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				assert.Nil(t, response["entity"])

				// TODO: check the record was actually created in the database
			},
		},
		{
			description: "POST log - wrong payload",
			query:       "POST /v1/logs",
			body:        `{"wrong": "payload"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Log","detail":"invalid format: message is required","status":422,"validationErrors":[{"field":"message","rule":"required","value":""}]}`,
		},
		{
			description: "POST log - wrong token",
			query:       "POST /v1/logs",
			body:        `{"message": "new log"}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "POST log with invalid JSON",
			query:       "POST /v1/logs",
			body:        `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Log`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: make this pass
		// {
		// 	query: "POST /v1/logs with JSON with extra fields",
		// 	body: `{"message": "new log", EXTRA_FIELD: "extra field not in schema"}`,
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 		"Content-Type":  {"application/json"},
		// 	},
		// 	expectedCode:        422,
		// 	expectedContentType: "application/problem+json",
		// 	validateFunc: func(t *testing.T, response map[string]interface{}) {
		// 		assert.Equal(t, `can't create Log`, response["title"])
		// 		assert.Equal(t, "invalid json", response["detail"])
		// 	},
		// },
		{
			description: "POST log with validation errors",
			query:       "POST /v1/logs",
			body:        `{"message": ""}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Log`, response["title"])
				assert.Equal(t, "invalid format: message is required", response["detail"])

				assert.IsType(t, []interface{}{}, response["validationErrors"])

				validationErrors := response["validationErrors"].([]interface{})
				assert.Equal(t, 1, len(validationErrors))

				firstValidationError := validationErrors[0].(map[string]interface{})

				for key := range firstValidationError {
					assert.Contains(t, []string{"field", "rule", "value"}, key)
				}
			},
		},
		{
			description: "POST log with empty body",
			query:       "POST /v1/logs",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Log`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: enforce this?
		// {
		// 	query: "POST /v1/logs with no Content-Type",
		// 	body:  "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// },
	}

	runTestCases(t, tests)
}

func TestWebhooksEndpoints(t *testing.T) {
	tests := []TestCase{
		// GET /webhooks/:id
		{
			query:               "GET /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			expectedCode:        200,
			expectedBody:        `{"id":"007bc84a-7e2d-43a0-b7e1-a256d4114aa7","url":"https://1-b.example.org/receiver","createdAt":"2017-05-01T00:00:00Z","updatedAt":"2017-05-01T00:00:00Z"}`,
			expectedContentType: "application/json",
		},
		{
			description:  "Non-existent webhook",
			query:        "GET /v1/webhooks/eea19c82-0449-11ed-bd84-d8bbc146d165",
			expectedCode: 404,
			expectedBody: `{"title":"can't get Webhook","detail":"Webhook was not found","status":404}`,

			expectedContentType: "application/problem+json",
		},

		// PATCH /webhooks/:id
		{
			query:    "PATCH /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			body:     `{"url": "https://new.example.org/receiver"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, "007bc84a-7e2d-43a0-b7e1-a256d4114aa7", response["id"])
				assert.Equal(t, "https://new.example.org/receiver", response["url"])
				assert.Equal(t, "2017-05-01T00:00:00Z", response["createdAt"])

				_, err := time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range response {
					assert.Contains(t, []string{"id", "url", "createdAt", "updatedAt"}, key)
				}
			},
		},
		{
			description: "PATCH webhook - wrong token",
			query:       "PATCH /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			body:        `{"url": "https://new.example.org/receiver"}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "PATCH /v1/webhooks with invalid JSON",
			query:       "PATCH /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			body:        `INVALID_JSON`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Webhook`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: make this pass
		// {
		// 	query: "PATCH /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7 with JSON with extra fields",
		// 	body: `{"url": "https://new.example.org/receiver", EXTRA_FIELD: "extra field not in schema"}`,
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 		"Content-Type":  {"application/json"},
		// 	},
		// 	expectedCode:        422,
		// 	expectedContentType: "application/problem+json",
		// 	validateFunc: func(t *testing.T, response map[string]interface{}) {
		// 		assert.Equal(t, `can't create Webhook`, response["title"])
		// 		assert.Equal(t, "invalid json", response["detail"])
		// 	},
		// },
		{
			description: "PATCH webhook with validation errors",
			query:       "PATCH /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			body:        `{"url": "INVALID_URL"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Webhook`, response["title"])
				assert.Equal(t, "invalid format: url is invalid", response["detail"])

				assert.IsType(t, []interface{}{}, response["validationErrors"])

				validationErrors := response["validationErrors"].([]interface{})
				assert.Equal(t, 1, len(validationErrors))

				firstValidationError := validationErrors[0].(map[string]interface{})

				for key := range firstValidationError {
					assert.Contains(t, []string{"field", "rule", "value"}, key)
				}
			},
		},
		{
			description: "PATCH /v1/webhooks with empty body",
			query:       "PATCH /v1/webhooks/007bc84a-7e2d-43a0-b7e1-a256d4114aa7",
			body:        "",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        400,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't update Webhook`, response["title"])
				assert.Equal(t, "invalid json", response["detail"])
			},
		},
		// TODO: enforce this?
		// {
		// 	query:    "PATCH /v1/webhooks with no Content-Type",
		// 	body:     "",
		// 	headers: map[string][]string{
		// 		"Authorization": {goodToken},
		// 	},
		// 	expectedCode:        404,
		// },

		// DELETE /webhooks/:id
		{
			description:         "Delete non-existent webhook",
			query:               "DELETE /v1/webhooks/NO_SUCH_WEBHOOK",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        404,
			// This error is different from because it's returned directly from Fiber's
			// route constraints, so we don't need to hit the database to find the resource
			// because we already know that's not a valid webhook id looking at its format.
			expectedBody:        `{"title":"Not Found","detail":"Cannot DELETE /v1/webhooks/NO_SUCH_WEBHOOK","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "DELETE webhook with bad authentication",
			query:       "DELETE /v1/webhooks/1702cd06-fffb-4d20-8f55-73e2a00ee052",
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
		{
			query:    "DELETE /v1/webhooks/24bc1b5d-fe81-47be-9d55-910f820bdd04",
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        204,
			expectedBody:        "",
			expectedContentType: "",
		},
	}

	runTestCases(t, tests)
}

func TestStatusEndpoints(t *testing.T) {
	tests := []TestCase{
		{
			query:               "GET /v1/status",
			expectedCode:        204,
			expectedBody:        "",
			expectedContentType: "",
			// TODO: test cache headers
		},
	}

	runTestCases(t, tests)
}

// TODO: test that webhooks are delivered
