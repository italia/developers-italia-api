package main

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogsEndpoints(t *testing.T) {
	tests := []TestCase{
		// GET /logs
		{
			query:               "GET /v1/logs",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 25, len(data))

				// Default pagination size is 25, so all the logs fit into a page
				// and cursors should be empty
				assertPaginationLinks(t, response, nil, "?page[after]=WyIyMDEwLTAxLTE1VDIzOjU5OjU5WiIsIjEyZjMwZDllLTA0MmUtMTFlZC04ZGRjLWQ4YmJjMTQ2ZDE2NSJd")

				var prevCreatedAt *time.Time = nil
				for _, item := range data {
					log := item

					assert.NotEmpty(t, log["message"])

					assertUUID(t, log["id"])

					createdAt := assertRFC3339(t, log["createdAt"])
					assertRFC3339(t, log["updatedAt"])

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

					assertOnlyKeys(t, log, "id", "createdAt", "updatedAt", "message", "entity")

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
				data := assertListResponse(t, response)

				assert.Equal(t, 3, len(data))

				assertPaginationLinks(t, response, nil, "?page[after]=WyIyMDEwLTA4LTAxVDIzOjU5OjU5WiIsIjRiNGExYjljLTA0MmUtMTFlZC04MmE4LWQ4YmJjMTQ2ZDE2NSJd")
			},
		},
		{
			description:         "GET with invalid format for page[size] query param",
			query:               "GET /v1/logs?page[size]=NOT_AN_INT",
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				assert.Equal(t, `can't get Logs`, response["title"])
				assert.Equal(t, "page[size] must be an integer", response["detail"])
			},
		},
		{
			description:         "GET with page[size] bigger than the max of 100 caps the size",
			query:               "GET /v1/logs?page[size]=200",
			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				items := assertListResponse(t, response)
				assert.Equal(t, 100, len(items))
			},
		},
		{
			description: `GET with "page[after]" query param`,
			query:       "GET /v1/logs?page[after]=WyIyMDEwLTA3LTAxVDIzOjU5OjU5WiIsIjg1MWZlMGY0LTA0MmUtMTFlZC05MzNlLWQ4YmJjMTQ2ZDE2NSJd",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := assertListResponse(t, response)

				assert.Equal(t, 18, len(data))

				assertPaginationLinks(t, response, "?page[before]=WyIyMDEwLTA2LTMwVDIzOjU5OjU5WiIsIjgyNTZmODgwLTA0MmUtMTFlZC04MmI5LWQ4YmJjMTQ2ZDE2NSJd", nil)
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
				data := assertListResponse(t, response)

				assert.Equal(t, 8, len(data))

				assertPaginationLinks(t, response, nil, "?page[after]=WyIyMDEwLTA3LTAxVDIzOjU5OjU5WiIsIjg1MWZlMGY0LTA0MmUtMTFlZC05MzNlLWQ4YmJjMTQ2ZDE2NSJd")
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
				data := assertListResponse(t, response)

				assert.Equal(t, 20, len(data))
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
			description: `GET with "search" query param`,
			query:       "GET /v1/logs?search=bad publiccode.yml",

			expectedCode:        200,
			expectedContentType: "application/json",
			validateFunc: func(t *testing.T, response map[string]interface{}) {
				data := response["data"].([]interface{})

				assert.Equal(t, 5, len(data))
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

				assertUUID(t, response["id"])
				assertTimestamps(t, response)

				assert.Nil(t, response["entity"])

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
			expectedBody:        `{"title":"can't create Log","detail":"unknown field in JSON input","status":422}`,
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
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
			},
		},
		{
			description: "POST /v1/logs with JSON with extra fields",
			query:       "POST /v1/logs",
			body:        `{"message": "new log", "EXTRA_FIELD": "extra field not in schema"}`,
			headers: map[string][]string{
				"Authorization": {goodToken},
				"Content-Type":  {"application/json"},
			},
			expectedCode:        422,
			expectedContentType: "application/problem+json",
			expectedBody:        `{"title":"can't create Log","detail":"unknown field in JSON input","status":422}`,
		},
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
				assert.Equal(t, "invalid or malformed JSON", response["detail"])
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

func TestLogsDBChecks(t *testing.T) {
	t.Run("POST log persists changes to DB", func(t *testing.T) {
		loadFixtures(t)

		const message = "New log persisted by DB check"

		req, err := newTestRequest("POST", "/v1/logs", strings.NewReader(`{"message":"`+message+`"}`))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)

		assert.Equal(t, 1, dbCount(t, "logs", "message", message))
	})

	t.Run("PATCH log persists changes to DB", func(t *testing.T) {
		loadFixtures(t)

		const (
			logID   = "4f95b0d0-042e-11ed-8253-d8bbc146d165"
			message = "Updated log message from DB check"
		)

		req, err := newTestRequest("PATCH", "/v1/logs/"+logID, strings.NewReader(`{"message":"`+message+`"}`))
		require.NoError(t, err)
		req.Header = map[string][]string{
			"Authorization": {goodToken},
			"Content-Type":  {"application/json"},
		}

		res, err := app.Test(req, -1)
		require.NoError(t, err)
		assert.Equal(t, 200, res.StatusCode)

		assert.Equal(t, message, dbValue(t, "logs", "message", "id", logID))
	})
}
