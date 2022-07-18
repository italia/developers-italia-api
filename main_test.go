package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"testing"

	"github.com/go-testfixtures/testfixtures/v3"
	"github.com/gofiber/fiber/v2"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

var (
	app       *fiber.App
	db        *sql.DB
	goodToken = "Bearer v2.local.TwwHUQEi8hr2Eo881_Bs5vK9dHOR5BgEU24QRf-U7VmUwI1yOEA6mFT0EsXioMkFT_T-jjrtIJ_Nv8f6hR6ifJXUOuzWEkm9Ijq1mqSjQatD3aDqKMyjjBA"
	badToken  = "Bearer v2.local.UngfrCDNwGUw4pff2oBNoyxYvOErcbVVqLndl6nzONafUCzktaOeMSmoI7B0h62zoxXXLqTm_Phl"
)

type TestCase struct {
	description string

	// Test input
	fixtures []string
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
	os.Remove("./test.db")

	os.Setenv("DATABASE_DSN", "file:./test.db")
	os.Setenv("ENVIRONMENT", "test")

	// echo -n 'test-paseto-key-dont-use-in-prod'  | base64
	os.Setenv("PASETO_KEY", "dGVzdC1wYXNldG8ta2V5LWRvbnQtdXNlLWluLXByb2Q=")

	var err error
	db, err = sql.Open("sqlite3", os.Getenv("DATABASE_DSN"))
	if err != nil {
		log.Fatal(err)
	}

	// This is needed, otherwise we get a database-locked error
	// TODO: investigate the root cause
	db.Exec("PRAGMA journal_mode=WAL;")

	// Setup the app as it is done in the main function
	app = Setup()
}

func TestMain(m *testing.M) {
	code := m.Run()

	os.Exit(code)
}

func loadFixtures(t *testing.T, files ...string) {
	var filesWithPath []string
	for _, file := range files {
		filesWithPath = append(filesWithPath, "test/testdata/fixtures/"+file)
	}

	fixtures, err := testfixtures.New(
		testfixtures.Database(db),
		testfixtures.Dialect("sqlite"),
		testfixtures.Files(filesWithPath...),
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
			if len(test.fixtures) > 0 {
				loadFixtures(t, test.fixtures...)
			}

			query := strings.Split(test.query, " ")

			req, _ := http.NewRequest(
				query[0],
				query[1],
				strings.NewReader(test.body),
			)
			if test.headers != nil {
				req.Header = test.headers
			}

			res, err := app.Test(req, -1)
			assert.Nil(t, err)

			assert.Equal(t, test.expectedCode, res.StatusCode)

			body, err := ioutil.ReadAll(res.Body)

			assert.Nil(t, err)

			if test.validateFunc != nil {
				var bodyMap map[string]interface{}
				err = json.Unmarshal(body, &bodyMap)
				assert.Nil(t, err)

				test.validateFunc(t, bodyMap)
			}

			if test.expectedBody != "" {
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
			expectedBody:        `{"title":"Not Found","status":404}`,
			expectedContentType: "application/problem+json",
		},
	}

	runTestCases(t, tests)
}

func TestPublishersEndpoints(t *testing.T) {
	tests := []TestCase{
		{
			query: "GET /v1/publishers",

			expectedCode:        200,
			expectedBody:        `{"data":[]}`,
			expectedContentType: "application/json",
		},
		{
			description:  "publishers get non-existing id",
			query:        "GET /v1/publishers/404",
			expectedCode: 404,
			expectedBody: `{"title":"can't get Publisher","detail":"Publisher was not found","status":404}`,

			expectedContentType: "application/problem+json",
		},
		{
			query: "POST /v1/publishers",
			body:  `{"URL":"https://www.example.com", "email":"example@example.com"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode: 200,
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				fMap := response["data"].(map[string]interface{})
				codeHost := fMap["codeHosting"].([]interface{})
				assert.Equal(t, 1, len(codeHost))
				codeHostElement := codeHost[0].(map[string]interface{})
				assert.Equal(t, codeHostElement["url"], "https://www.example.com")
				assert.Equal(t, fMap["email"], "example@example.com")
			},
			expectedContentType: "application/json",
		},
		{
			description: "POST publisher - wrong token",
			query:       "POST /v1/publishers",
			body:        `{"name": "New publisher"}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
	}

	runTestCases(t, tests)
}

func TestX(t *testing.T) {
	tests := []TestCase{
		{
			query:    "GET /v1/logs",
			fixtures: []string{"logs.yml"},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 10, len(data))
			},
		},
	}
	runTestCases(t, tests)
}

func TestLogsEndpoints(t *testing.T) {
	tests := []TestCase{
		{
			query:    "GET /v1/logs",
			fixtures: []string{"logs.yml"},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 10, len(data))
			},
		},
		{
			query:    "GET /v1/logs",
			fixtures: []string{"logs.yml"},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 10, len(data))
			},
		},
		{
			query:        "GET /v1/logs/eea19c82-0449-11ed-bd84-d8bbc146d165",
			expectedCode: 404,
			expectedBody: `{"title":"can't get Log","detail":"Log was not found","status":404}`,

			expectedContentType: "application/problem+json",
		},
		{
			query: "POST /v1/logs",
			body:  `{"message": "New log"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
		},
		{
			description: "POST log - wrong token",
			query:       "POST /v1/logs",
			body:        `{"message": "New log"}`,
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedBody:        `{"title":"token authentication failed","status":401}`,
			expectedContentType: "application/problem+json",
		},
	}

	runTestCases(t, tests)
}

func TestStatusEndpoints(t *testing.T) {
	tests := []TestCase{
		{
			query:               "GET /status",
			expectedCode:        204,
			expectedBody:        "",
			expectedContentType: "",
			// TODO: test cache headers
		},
	}

	runTestCases(t, tests)
}
