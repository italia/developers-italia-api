package main

import (
	"database/sql"
	"encoding/json"
	"io/ioutil"
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
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
)

const UUID_REGEXP = "[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}"

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

			u, _ := url.Parse(query[1])

			req, _ := http.NewRequest(
				query[0],
				query[1],
				strings.NewReader(test.body),
			)
			if test.headers != nil {
				req.Header = test.headers
			}
			req.URL.RawQuery = u.Query().Encode()

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
			expectedBody:        `{"title":"Not Found","detail":"Cannot GET /v1/i-dont-exist","status":404}`,
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

func TestSoftwareEndpoints(t *testing.T) {
	tests := []TestCase{
		// GET /software
		{
			query:    "GET /v1/software",
			fixtures: []string{"software.yml", "software_urls.yml"},

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
				assert.Equal(t, "?page[after]=WyJjMzUzNzU2ZS04NTk3LTRlNDYtYTk5Yi03ZGEyZTE0MTYwM2IiLCIyMDE0LTA1LTAxVDAwOjAwOjAwWiJd", links["next"])

				assert.IsType(t, map[string]interface{}{}, data[0])
				firstSoftware := data[0].(map[string]interface{})
				assert.NotEmpty(t, firstSoftware["publiccodeYml"])

				assert.IsType(t, []interface{}{}, firstSoftware["urls"])
				assert.Greater(t, len(firstSoftware["urls"].([]interface{})), 0)

				match, err := regexp.MatchString(UUID_REGEXP, firstSoftware["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, firstSoftware["createdAt"].(string))
				assert.Nil(t, err)
				_, err = time.Parse(time.RFC3339, firstSoftware["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range firstSoftware {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "urls", "publiccodeYml"}, key)
				}
			},
		},
		{
			description: "GET with page[size] query param",
			query:       "GET /v1/software?page[size]=2",
			fixtures:    []string{"software.yml"},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 2, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyIxMjQyODBkNy03NTUyLTRmZmUtOTM5Zi1mNDY2OTdjYzBlOGEiLCIyMDE1LTA0LTI2VDAwOjAwOjAwWiJd", links["next"])
			},
		},
		// TODO
		// {
		// 	description: "GET with invalid format for page[size] query param",
		// 	query:    "GET /v1/software?page[size]=NOT_AN_INT",
		// 	fixtures: []string{"software.yml"},

		// 	expectedCode:        422,
		// 	expectedContentType: "application/json",
		// },
		// TODO
		// {
		// 	description: "GET with page[size] bigger than the max of 100",
		// 	query:    "GET /v1/software?page[size]=200",
		// 	fixtures: []string{"software.yml", "software_urls.yml"},

		// 	expectedCode:        422,
		// 	expectedContentType: "application/json",
		// },
		{
			description: `GET with "page[after]" query param`,
			query:       "GET /v1/software?page[after]=WyJjMzUzNzU2ZS04NTk3LTRlNDYtYTk5Yi03ZGEyZTE0MTYwM2IiLCIyMDE0LTA1LTAxVDAwOjAwOjAwWiJd",
			fixtures:    []string{"software.yml"},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 5, len(data))

				links := response["links"].(map[string]interface{})
				assert.Equal(t, "?page[before]=WyJjNWRlYzZmYS04YTAxLTQ4ODEtOWU3ZC0xMzI3NzBkNDIxNGQiLCIyMDE1LTAyLTI1VDAwOjAwOjAwWiJd", links["prev"])
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
			query:       "GET /v1/software?page[before]=WyJjNWRlYzZmYS04YTAxLTQ4ODEtOWU3ZC0xMzI3NzBkNDIxNGQiLCIyMDE1LTAyLTI1VDAwOjAwOjAwWiJd",
			fixtures:    []string{"software.yml"},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 25, len(data))

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyJjMzUzNzU2ZS04NTk3LTRlNDYtYTk5Yi03ZGEyZTE0MTYwM2IiLCIyMDE0LTA1LTAxVDAwOjAwOjAwWiJd", links["next"])
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
			fixtures:    []string{"software.yml"},

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
			fixtures:    []string{"software.yml"},

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
			fixtures:    []string{"software.yml"},

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
			fixtures:    []string{"software.yml"},

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
			fixtures:            []string{"software.yml", "software_urls.yml"},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.NotEmpty(t, response["publiccodeYml"])

				assert.IsType(t, []interface{}{}, response["urls"])
				assert.Greater(t, len(response["urls"].([]interface{})), 0)

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)
				_, err = time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range response {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "urls", "publiccodeYml"}, key)
				}
			},
		},

		// POST /software
		{
			query: "POST /v1/software",
			body:  `{"publiccodeYml": "-", "urls": ["https://software.example.org"]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["urls"])
				assert.Equal(t, len(response["urls"].([]interface{})), 1)

				// TODO: check urls content
				assert.NotEmpty(t, response["publiccodeYml"])

				match, err := regexp.MatchString(UUID_REGEXP, response["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, response["createdAt"].(string))
				assert.Nil(t, err)

				_, err = time.Parse(time.RFC3339, response["updatedAt"].(string))
				assert.Nil(t, err)

				// TODO: check the record was actually created in the database
				// TODO: check there are no dangling software_urls
			},
		},
		{
			description: "POST software - wrong token",
			query:       "POST /v1/software",
			body:        `{"publiccodeYml":  "-", "urls": ["https://software.example.org"]}`,
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
			description: "POST /v1/software with validation errors",
			query:       "POST /v1/software",
			body:        `{"message": ""}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Software`, response["title"])
				assert.Equal(t, "invalid format", response["detail"])

				assert.IsType(t, []interface{}{}, response["validationErrors"])

				validationErrors := response["validationErrors"].([]interface{})
				assert.Equal(t, len(validationErrors), 2)

				firstValidationError := validationErrors[0].(map[string]interface{})

				for key := range firstValidationError {
					assert.Contains(t, []string{"field", "rule", "providedValue"}, key)
				}
			},
		},
		{
			description: "POST /v1/software with empty body",
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
			body:  `{"publiccodeYml": "publiccodedata", "urls": ["https://software.example.org", "https://sofware-old.example.com", "https://software-old.example.org"]}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			fixtures: []string{"software.yml", "software_urls.yml"},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["urls"])
				assert.Equal(t, len(response["urls"].([]interface{})), 3)
				// TODO: check urls content

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
			description: "PATCH /v1/software with invalid JSON",
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
		// 	description: "PATCH /v1/software with JSON with extra fields",
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
			description: "POST /v1/software with validation errors",
			query:       "POST /v1/software",
			body:        `{"message": ""}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't create Software`, response["title"])
				assert.Equal(t, "invalid format", response["detail"])

				assert.IsType(t, []interface{}{}, response["validationErrors"])

				validationErrors := response["validationErrors"].([]interface{})
				assert.Equal(t, len(validationErrors), 2)

				firstValidationError := validationErrors[0].(map[string]interface{})

				for key := range firstValidationError {
					assert.Contains(t, []string{"field", "rule", "providedValue"}, key)
				}
			},
		},
		{
			description: "POST /v1/software with empty body",
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

		// DELETE /software/:id
		{
			description:         "Delete non-existent software",
			query:               "GET /v1/software/eea19c82-0449-11ed-bd84-d8bbc146d165",
			expectedCode:        404,
			expectedBody:        `{"title":"can't get Software","detail":"Software was not found","status":404}`,
			expectedContentType: "application/problem+json",
		},
		{
			description: "DELETE software with bad authentication",
			query:       "DELETE /v1/software/11e101c4-f989-4cc4-a665-63f9f34e83f6",
			fixtures:    []string{"software.yml", "software_urls.yml"},
			headers: map[string][]string{
				"Authorization": {badToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        401,
			expectedContentType: "application/problem+json",
		},
		{
			query:    "DELETE /v1/software/11e101c4-f989-4cc4-a665-63f9f34e83f6",
			fixtures: []string{"software.yml", "software_urls.yml"},
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        204,
			expectedBody:        "",
			expectedContentType: "application/json",
		},

		// GET /software/:id/logs
		{
			query:    "GET /v1/software/c353756e-8597-4e46-a99b-7da2e141603b/logs",
			fixtures: []string{"software.yml", "logs.yml"},

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

				assert.IsType(t, map[string]interface{}{}, data[0])
				firstLog := data[0].(map[string]interface{})
				assert.NotEmpty(t, firstLog["message"])

				match, err := regexp.MatchString(UUID_REGEXP, firstLog["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, firstLog["createdAt"].(string))
				assert.Nil(t, err)
				_, err = time.Parse(time.RFC3339, firstLog["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range firstLog {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "message", "entity"}, key)
				}

				// TODO assert.NotEmpty(t, firstLog["entity"])
			},
		},
		{
			description: "GET /v1/software/:id/logs for non existing software",
			query:       "GET /v1/software/NO_SUCH_SOFTWARE/logs",
			fixtures:    []string{"software.yml", "logs.yml"},

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
			fixtures:    []string{"logs.yml"},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 2, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyIxOGE3MDM2Mi0wNDJlLTExZWQtYjc5My1kOGJiYzE0NmQxNjUiLCIyMDEwLTAxLTMwVDIzOjU5OjU5WiJd", links["next"])
			},
		},

		// POST /software/:id/logs
		{
			description: "POST /v1/software/:id/logs for non existing software",
			query:       "POST /v1/software/NO_SUCH_SOFTWARE/logs",
			fixtures:    []string{"software.yml", "logs.yml"},
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
			fixtures: []string{"software.yml", "logs.yml"},
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
			description: "POST /v1/logs with invalid JSON",
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
			description: "POST /v1/logs with validation errors",
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
				assert.Equal(t, "invalid format", response["detail"])

				assert.IsType(t, []interface{}{}, response["validationErrors"])

				validationErrors := response["validationErrors"].([]interface{})
				assert.Equal(t, len(validationErrors), 1)

				firstValidationError := validationErrors[0].(map[string]interface{})

				for key := range firstValidationError {
					assert.Contains(t, []string{"field", "rule", "providedValue"}, key)
				}
			},
		},
		{
			description: "POST /v1/logs with empty body",
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
	}

	runTestCases(t, tests)
}

func TestLogsEndpoints(t *testing.T) {
	tests := []TestCase{
		// GET /logs
		{
			query:    "GET /v1/logs",
			fixtures: []string{"logs.yml"},

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

				assert.IsType(t, map[string]interface{}{}, data[0])
				firstLog := data[0].(map[string]interface{})
				assert.NotEmpty(t, firstLog["message"])

				match, err := regexp.MatchString(UUID_REGEXP, firstLog["id"].(string))
				assert.Nil(t, err)
				assert.True(t, match)

				_, err = time.Parse(time.RFC3339, firstLog["createdAt"].(string))
				assert.Nil(t, err)
				_, err = time.Parse(time.RFC3339, firstLog["updatedAt"].(string))
				assert.Nil(t, err)

				for key := range firstLog {
					assert.Contains(t, []string{"id", "createdAt", "updatedAt", "message", "entity"}, key)
				}

				// TODO assert.NotEmpty(t, firstLog["entity"])
			},
		},
		{
			description: "GET with page[size] query param",
			query:       "GET /v1/logs?page[size]=3",
			fixtures:    []string{"logs.yml"},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 3, len(data))

				assert.IsType(t, map[string]interface{}{}, response["links"])

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyIyZGZiMmJjMi0wNDJkLTExZWQtOTMzOC1kOGJiYzE0NmQxNjUiLCIyMDEwLTAxLTAxVDIzOjU5OjU5WiJd", links["next"])
			},
		},
		// TODO
		// {
		// 	description: "GET with invalid format for page[size] query param",
		// 	query:    "GET /v1/logs?page[size]=NOT_AN_INT",
		// 	fixtures: []string{"logs.yml"},

		// 	expectedCode:        422,
		// 	expectedContentType: "application/json",
		// },
		{
			description: `GET with "page[after]" query param`,
			query:       "GET /v1/logs?page[after]=WyI0Zjk1YjBkMC0wNDJlLTExZWQtODI1My1kOGJiYzE0NmQxNjUiLCIyMDEwLTAyLTAxVDIzOjU5OjU5WiJd",
			fixtures:    []string{"logs.yml"},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 17, len(data))

				links := response["links"].(map[string]interface{})
				assert.Equal(t, "?page[before]=WyI1MzY1MDUwOC0wNDJlLTExZWQtOWI4NC1kOGJiYzE0NmQxNjUiLCIyMDEwLTAyLTE1VDIzOjU5OjU5WiJd", links["prev"])
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
			query:       "GET /v1/logs?page[before]=WyI0Zjk1YjBkMC0wNDJlLTExZWQtODI1My1kOGJiYzE0NmQxNjUiLCIyMDEwLTEyLTMxVDIzOjU5OjU5WiJd",
			fixtures:    []string{"logs.yml"},

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.IsType(t, []interface{}{}, response["data"])
				data := response["data"].([]interface{})

				assert.Equal(t, 4, len(data))

				links := response["links"].(map[string]interface{})
				assert.Nil(t, links["prev"])
				assert.Equal(t, "?page[after]=WyI0Zjk1YjBkMC0wNDJlLTExZWQtODI1My1kOGJiYzE0NmQxNjUiLCIyMDEwLTAyLTAxVDIzOjU5OjU5WiJd", links["next"])
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
			fixtures:    []string{"logs.yml"},

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
			fixtures:    []string{"logs.yml"},

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
			fixtures:    []string{"logs.yml"},

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
			fixtures:    []string{"logs.yml"},

			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Logs`, response["title"])
				assert.Equal(t, "invalid date time format (RFC 3339 needed)", response["detail"])
			},
		},
		{
			description:  "Non-existent log",
			query:        "GET /v1/logs/eea19c82-0449-11ed-bd84-d8bbc146d165",
			expectedCode: 404,
			expectedBody: `{"title":"can't get Log","detail":"Log was not found","status":404}`,

			expectedContentType: "application/problem+json",
		},

		// POST /logs
		{
			query: "POST /v1/logs",
			body:  `{"message": "New log from test suite"}`,
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

				// TODO: check the record was actually created in the database
			},
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
			description: "POST /v1/logs with invalid JSON",
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
			description: "POST /v1/logs with validation errors",
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
				assert.Equal(t, "invalid format", response["detail"])

				assert.IsType(t, []interface{}{}, response["validationErrors"])

				validationErrors := response["validationErrors"].([]interface{})
				assert.Equal(t, len(validationErrors), 1)

				firstValidationError := validationErrors[0].(map[string]interface{})

				for key := range firstValidationError {
					assert.Contains(t, []string{"field", "rule", "providedValue"}, key)
				}
			},
		},
		{
			description: "POST /v1/logs with empty body",
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
